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
//
// Os formatos antigos com região continuam aceitos por
// compatibilidade, porém todos executam a mesma varredura:
//
//	!coletar floresta
//	!coletar pedreira
//	!coletar mina
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

	if len(parts) == 2 {
		if _, ok :=
			parseNaturalRPGRegion(
				parts[1],
			); !ok {

			sendRPGGatheringUsage(
				client,
				msgEvent,
			)

			return
		}
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

	_, err =
		rpg.ClaimStarterSet(
			groupJID,
			jid,
			catalog,
		)

	switch {
	case err == nil:

	case errors.Is(
		err,
		rpg.ErrStarterAlreadyClaimed,
	):

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

	result, err :=
		rpg.GatherAllRegions(
			groupJID,
			jid,
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
					"⏳ *SUA ROTA DE COLETA AINDA ESTÁ EM DESCANSO*\n\n"+
						"Nova varredura disponível em *%s*.",
					formatGatheringDuration(
						cooldownErr.Remaining,
					),
				),
			)

		case errors.Is(
			err,
			rpg.ErrNoGatheringMaterials,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Nenhum recurso pôde ser encontrado durante a varredura.",
			)

		default:
			sendRPGInternalError(
				client,
				msgEvent,
				"realizando varredura de coleta",
				err,
			)
		}

		return
	}

	var builder strings.Builder

	builder.WriteString(
		"🌿 *VARREDURA DE RECURSOS CONCLUÍDA*\n",
	)

	for _, region := range result.Regions {

		fmt.Fprintf(
			&builder,
			"\n%s *%s*\n",
			region.Region.Icon(),
			region.Region.Name(),
		)

		for _, drop := range region.Drops {

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

		if region.BonusRoll {
			builder.WriteString(
				"🍀 Exploração extra nesta região!\n",
			)
		}
	}

	fmt.Fprintf(
		&builder,
		"\n📦 Total: *%d recursos* • 🎲 %d rolagens",
		result.TotalItems,
		result.Rolls,
	)

	if result.EpicItems > 0 {
		fmt.Fprintf(
			&builder,
			"\n🟣 Épicos coletados: *%d*",
			result.EpicItems,
		)
	}

	if result.LegendaryItems > 0 {
		fmt.Fprintf(
			&builder,
			"\n🔴 Lendários coletados: *%d*",
			result.LegendaryItems,
		)
	}

	fmt.Fprintf(
		&builder,
		"\n\n⏳ Nova varredura em *%s*.",
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
			"Use *!coletar* para fazer uma varredura completa em:\n"+
			"🌲 Floresta\n"+
			"🪨 Pedreira\n"+
			"⛏️ Mina\n\n"+
			"⏳ Cooldown: *30 segundos*.\n\n"+
			"🌱 Aliases: *!coleta • !recolher*\n\n"+
			"💡 Os antigos comandos por região continuam aceitos, "+
			"mas agora também executam a varredura completa.",
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
