package handler

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// handleRPGGathering processa:
//
//	!coletar
//	!coletar floresta
//	!coletar pedreira
//	!coletar mina
//
// Sem região informada, uma região é escolhida aleatoriamente.
func handleRPGGathering(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {
	if len(parts) > 2 {
		sendRPGGatheringUsage(
			client,
			msgEvent,
		)

		return
	}

	var region rpg.GatheringRegion

	if len(parts) == 2 {
		parsedRegion, ok :=
			parseNaturalRPGRegion(
				parts[1],
			)

		if !ok {
			sendRPGGatheringUsage(
				client,
				msgEvent,
			)

			return
		}

		region = parsedRegion
	}

	catalog,
		materialCatalog,
		err :=
		getRPGCatalogs()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo para coleta",
			err,
		)

		return
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	// ======================================================
	// MIGRAÇÃO RPG
	// ======================================================
	//
	// Jogadores antigos que já possuem carteira Gold,
	// mas ainda não possuem RPG, recebem somente o
	// Set do Recruta.
	//
	// Nenhum Gold inicial é concedido por !coletar.
	// ======================================================

	_, err =
		rpg.ClaimStarterSet(
			groupJID,
			jid,
			catalog,
		)

	switch {
	case err == nil:
		// Starter criado agora.

	case errors.Is(
		err,
		rpg.ErrStarterAlreadyClaimed,
	):
		// RPG já inicializado.

	case errors.Is(
		err,
		rpg.ErrWalletNotFound,
	):
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Você ainda não possui uma carteira Gold neste grupo.\n\n"+
				"Use *!gold* ou *!rpg* para iniciar sua jornada antes de coletar recursos.",
		)

		return

	default:
		sendRPGInternalError(
			client,
			msgEvent,
			"preparando aventureiro para coleta",
			err,
		)

		return
	}

	// ======================================================
	// COLETA
	// ======================================================

	result, err :=
		rpg.Gather(
			groupJID,
			jid,
			region,
			materialCatalog,
		)

	if err != nil {
		var cooldownErr *rpg.GatheringCooldownError

		switch {
		case errors.As(
			err,
			&cooldownErr,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"⏳ %s *%s ainda está em cooldown.*\n\n"+
						"Tente explorar esta região novamente em *%s*.\n\n"+
						"💡 As outras regiões continuam disponíveis.",
					cooldownErr.Region.Icon(),
					cooldownErr.Region.Name(),
					formatGatheringDuration(
						cooldownErr.Remaining,
					),
				),
			)

		case errors.Is(
			err,
			rpg.ErrInvalidGatheringRegion,
		):
			sendRPGGatheringUsage(
				client,
				msgEvent,
			)

		case errors.Is(
			err,
			rpg.ErrNoGatheringMaterials,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Nenhum recurso pôde ser encontrado nesta região.",
			)

		default:
			sendRPGInternalError(
				client,
				msgEvent,
				"realizando coleta",
				err,
			)
		}

		return
	}

	// ======================================================
	// RESPOSTA
	// ======================================================

	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"%s *%s*\n",
		result.Region.Icon(),
		result.Region.Name(),
	)

	for _, drop := range result.Drops {
		fmt.Fprintf(
			&builder,
			"%s %s ×%d\n",
			rpgGatheringRarityIcon(
				drop.Material.Rarity,
			),
			drop.Material.Name,
			drop.Quantity,
		)
	}

	fmt.Fprintf(
		&builder,
		"\n⏳ %s",
		formatGatheringDuration(
			rpg.GatheringCooldown,
		),
	)

	builder.WriteString(
		recordRPGXP(
			groupJID,
			jid,
			RPGGatheringXPReward,
		),
	)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		builder.String(),
	)
}

func sendRPGGatheringUsage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"🌿 *COLETA DE RECURSOS*\n\n"+
			"Use:\n"+
			"*!coletar*\n"+
			"*!coletar floresta*\n"+
			"*!coletar pedreira*\n"+
			"*!coletar mina*\n\n"+
			"🌱 Aliases:\n"+
			"*!coleta • !recolher*\n\n"+
			"🗺️ Regiões também aceitam:\n"+
			"*mata • bosque • rocha • mineração*",
	)
}

func formatGatheringDuration(
	duration time.Duration,
) string {
	if duration <= 0 {
		return "alguns segundos"
	}

	seconds :=
		int(
			duration.Seconds(),
		)

	if duration%time.Second != 0 {
		seconds++
	}

	if seconds < 1 {
		seconds = 1
	}

	minutes :=
		seconds / 60

	seconds =
		seconds % 60

	if minutes <= 0 {
		return fmt.Sprintf(
			"%d segundos",
			seconds,
		)
	}

	if seconds == 0 {
		return fmt.Sprintf(
			"%d minutos",
			minutes,
		)
	}

	return fmt.Sprintf(
		"%d min %d s",
		minutes,
		seconds,
	)
}

func rpgGatheringRarityIcon(
	rarity rpg.Rarity,
) string {
	switch rarity {
	case rpg.RarityWorn:
		return "⚪"

	case rpg.RarityCommon:
		return "🟢"

	case rpg.RarityRare:
		return "🔵"

	case rpg.RarityEpic:
		return "🟣"

	case rpg.RarityLegendary:
		return "🔴"

	case rpg.RarityMythic:
		return "🟠"

	case rpg.RaritySacred:
		return "🌟"

	default:
		return "📦"
	}
}
