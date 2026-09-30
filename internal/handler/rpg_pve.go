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

// handleRPGPVE processa:
//
//	!pve
//	!pve floresta
//	!pve pedreira
//	!pve mina
//
// Sem região, uma região é escolhida aleatoriamente.
func handleRPGPVE(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {

	if tryHandleRPGPVEBatch(
		client,
		msgEvent,
		parts,
	) {
		return
	}

	if len(parts) > 2 {
		sendRPGPVEUsage(
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
			sendRPGPVEUsage(
				client,
				msgEvent,
			)
			return
		}

		region = parsedRegion
	}

	itemCatalog,
		materialCatalog,
		err :=
		getRPGCatalogs()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo RPG para PvE",
			err,
		)
		return
	}

	enemyCatalog, err :=
		rpg.LoadEnemyCatalog()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo de monstros",
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

	// Garante que jogadores antigos com carteira Gold,
	// mas ainda sem personagem RPG completo, recebam
	// o Set do Recruta antes do primeiro PvE.
	_, err =
		rpg.ClaimStarterSet(
			groupJID,
			jid,
			itemCatalog,
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
				"Use *!gold* ou *!rpg* para iniciar sua jornada antes de enfrentar monstros.",
		)
		return

	default:
		sendRPGInternalError(
			client,
			msgEvent,
			"preparando aventureiro para PvE",
			err,
		)
		return
	}

	result, err :=
		rpg.PlayPVE(
			groupJID,
			jid,
			region,
			itemCatalog,
			materialCatalog,
			enemyCatalog,
		)

	if err != nil {
		var cooldownErr *rpg.PVECooldownError

		switch {
		case errors.As(
			err,
			&cooldownErr,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"⏳ Próximo PvE em *%s*.",
					formatPVEDuration(
						cooldownErr.Remaining,
					),
				),
			)

		case errors.Is(
			err,
			rpg.ErrInvalidPVERegion,
		):
			sendRPGPVEUsage(
				client,
				msgEvent,
			)

		case errors.Is(
			err,
			rpg.ErrInvalidPVEPlayerPower,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Seu Poder de Combate não é suficiente para iniciar um encontro PvE.\n\n"+
					"Equipe seus itens com *!equipar* e tente novamente.",
			)

		default:
			sendRPGInternalError(
				client,
				msgEvent,
				"realizando encontro PvE",
				err,
			)
		}

		return
	}

	response :=
		renderPVEResult(
			result,
		)

	if result.EquipmentDrop != nil {
		autoEquip,
			autoEquipErr :=
			rpg.AutoEquipIfBetter(
				groupJID,
				jid,
				result.EquipmentDrop.ID,
				itemCatalog,
			)

		if autoEquipErr == nil &&
			autoEquip != nil &&
			autoEquip.Equipped {

			response +=
				fmt.Sprintf(
					"\n⚡ Autoequip: *%s*",
					result.EquipmentDrop.Name,
				)
		}
	}

	if result.Won {
		response +=
			recordRPGXP(
				groupJID,
				jid,
				rpgPVEVictoryXP(
					result,
				),
			)
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)

	notifyForgeReadyRecipes(
		client,
		msgEvent,
		groupJID,
		jid,
		itemCatalog,
		materialCatalog,
	)
}

func renderPVEResult(
	result *rpg.PVEResult,
) string {
	var builder strings.Builder

	rarityIcon,
		_ :=
		formatPVERarity(
			result.Enemy.Rarity,
		)

	fmt.Fprintf(
		&builder,
		"⚔️ %s *%s*",
		rarityIcon,
		result.Enemy.Name,
	)

	if !result.Won {
		builder.WriteString(
			" • 💀",
		)

		return builder.String()
	}

	if result.Enemy.Boss {
		builder.WriteString(
			" • 👑",
		)
	} else {
		builder.WriteString(
			" • ✅",
		)
	}

	if result.GoldReward > 0 ||
		result.CrystalReward > 0 {

		fmt.Fprintf(
			&builder,
			"\n💰 %s • 💎 %s",
			formatGold(
				result.GoldReward,
			),
			formatRPGNumber(
				result.CrystalReward,
			),
		)
	}

	if result.MaterialDrop != nil {
		fmt.Fprintf(
			&builder,
			"\n📦 %s×%d",
			result.MaterialDrop.Material.Name,
			result.MaterialDrop.Quantity,
		)
	}

	if result.EquipmentDrop != nil {
		fmt.Fprintf(
			&builder,
			"\n🎁 %s %s",
			rpgRarityIcon(
				result.EquipmentDrop.Rarity,
			),
			result.EquipmentDrop.Name,
		)
	}

	if result.BossKill &&
		result.Blessing != nil {

		fmt.Fprintf(
			&builder,
			"\n✨ %s",
			result.Blessing.Name,
		)
	}

	return strings.TrimSpace(
		builder.String(),
	)
}

func formatPVERarity(
	rarity rpg.Rarity,
) (
	string,
	string,
) {
	switch rarity {
	case rpg.RarityCommon:
		return "⚪", "COMUM"

	case rpg.RarityRare:
		return "🔵", "RARO"

	case rpg.RarityEpic:
		return "🟣", "ÉPICO"

	case rpg.RarityLegendary:
		return "🟠", "LENDÁRIO"
	}

	return "👹", string(rarity)
}

func formatPVEDuration(
	duration time.Duration,
) string {
	if duration < 0 {
		duration = 0
	}

	seconds :=
		int(
			(duration +
				time.Second -
				1) /
				time.Second,
		)

	if seconds < 60 {
		if seconds == 1 {
			return "1 segundo"
		}

		return fmt.Sprintf(
			"%d segundos",
			seconds,
		)
	}

	minutes :=
		seconds / 60

	remainingSeconds :=
		seconds % 60

	if remainingSeconds == 0 {
		if minutes == 1 {
			return "1 minuto"
		}

		return fmt.Sprintf(
			"%d minutos",
			minutes,
		)
	}

	return fmt.Sprintf(
		"%dm %ds",
		minutes,
		remainingSeconds,
	)
}

func sendRPGPVEUsage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"🐺 *CAÇADAS PvE*\n\n"+
			"Use:\n"+
			"*!pve*\n"+
			"*!pve floresta*\n"+
			"*!pve pedreira*\n"+
			"*!pve mina*\n\n"+
			"🚀 Expedições em lote:\n"+
			"*!pve 100*\n"+
			"*!pve mina 500*\n"+
			"*!pve floresta 1000*\n"+
			"Máximo: *1.000* batalhas.\n\n"+
			"⚔️ Aliases:\n"+
			"*!caçar • !lutar • !combater*\n\n"+
			"🗺️ Regiões também aceitam:\n"+
			"*mata • bosque • rocha • mineração*",
	)
}
