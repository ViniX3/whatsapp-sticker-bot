package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func tryHandleRPGPVEBatch(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) bool {
	region, battles, isBatch, valid :=
		parseRPGPVEBatchArguments(parts)

	if !isBatch {
		return false
	}

	if !valid {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"❌ Quantidade inválida para expedição PvE.\n\n"+
					"Use de *1 até %d* batalhas.\n\n"+
					"Exemplos:\n"+
					"*!pve 100*\n"+
					"*!pve mina 500*\n"+
					"*!pve floresta 1000*",
				rpg.PVEBatchMaxBattles,
			),
		)
		return true
	}

	itemCatalog, materialCatalog, err := getRPGCatalogs()
	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo RPG para expedição PvE",
			err,
		)
		return true
	}

	enemyCatalog, err := rpg.LoadEnemyCatalog()
	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo de monstros para expedição PvE",
			err,
		)
		return true
	}

	groupJID := msgEvent.Info.Chat.String()
	jid := canonicalSenderJID(msgEvent)

	_, err = rpg.ClaimStarterSet(groupJID, jid, itemCatalog)
	switch {
	case err == nil:
	case errors.Is(err, rpg.ErrStarterAlreadyClaimed):
	case errors.Is(err, rpg.ErrWalletNotFound):
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Você ainda não possui uma carteira Gold neste grupo.\n\n"+
				"Use *!gold* ou *!rpg* para iniciar sua jornada antes da expedição.",
		)
		return true
	default:
		sendRPGInternalError(
			client,
			msgEvent,
			"preparando aventureiro para expedição PvE",
			err,
		)
		return true
	}

	result, err := rpg.PlayPVEBatch(
		groupJID,
		jid,
		region,
		battles,
		itemCatalog,
		materialCatalog,
		enemyCatalog,
	)
	if err != nil {
		var cooldownErr *rpg.PVEBatchCooldownError

		switch {
		case errors.As(err, &cooldownErr):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"⏳ Você ainda está se recuperando da última expedição.\n\n"+
						"Nova expedição PvE em *%s*.\n"+
						"⚔️ O *!pve* comum continua disponível.",
					formatPVEDuration(cooldownErr.Remaining),
				),
			)
		case errors.Is(err, rpg.ErrInvalidPVEBatchAmount):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"❌ Use entre *1 e %d* batalhas por expedição.",
					rpg.PVEBatchMaxBattles,
				),
			)
		case errors.Is(err, rpg.ErrInvalidPVERegion):
			sendRPGPVEUsage(client, msgEvent)
		case errors.Is(err, rpg.ErrInvalidPVEPlayerPower):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Seu Poder de Combate não é suficiente para iniciar uma expedição PvE.\n\n"+
					"Equipe seus itens com *!equipar* e tente novamente.",
			)
		default:
			sendRPGInternalError(
				client,
				msgEvent,
				"realizando expedição PvE",
				err,
			)
		}
		return true
	}

	autoEquipped :=
		autoEquipPVEBatchDrops(
			groupJID,
			jid,
			result,
			itemCatalog,
		)

	totalXP := 0
	for index := range result.VictoryResults {
		totalXP += rpgPVEVictoryXP(&result.VictoryResults[index])
	}

	response :=
		renderPVEBatchResult(
			result,
			autoEquipped,
		)
	if totalXP > 0 {
		response += recordRPGXP(groupJID, jid, totalXP)
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

	return true
}

func parseRPGPVEBatchArguments(
	parts []string,
) (rpg.GatheringRegion, int, bool, bool) {
	if len(parts) == 2 {
		battles, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", 0, false, false
		}
		return "",
			battles,
			true,
			battles >= 1 &&
				battles <= rpg.PVEBatchMaxBattles
	}

	if len(parts) == 3 {
		battles, err := strconv.Atoi(parts[2])
		if err != nil {
			return "", 0, false, false
		}

		region, ok := parseNaturalRPGRegion(parts[1])
		if !ok {
			return "", battles, true, false
		}

		return region,
			battles,
			true,
			battles >= 1 &&
				battles <= rpg.PVEBatchMaxBattles
	}

	return "", 0, false, false
}

func autoEquipPVEBatchDrops(
	groupJID string,
	jid string,
	result *rpg.PVEBatchResult,
	catalog *rpg.Catalog,
) []string {
	if result == nil || catalog == nil {
		return nil
	}

	equipped := make([]string, 0, 3)

	// result.Equipment já vem ordenado por raridade/poder.
	// Assim o melhor item de cada slot é avaliado primeiro.
	for _, drop := range result.Equipment {
		autoEquip, err :=
			rpg.AutoEquipIfBetter(
				groupJID,
				jid,
				drop.Item.ID,
				catalog,
			)

		if err != nil || autoEquip == nil || !autoEquip.Equipped {
			continue
		}

		equipped = append(equipped, drop.Item.Name)

		if len(equipped) >= 3 {
			break
		}
	}

	return equipped
}

func renderPVEBatchResult(
	result *rpg.PVEBatchResult,
	autoEquipped []string,
) string {
	var builder strings.Builder

	regionName := "Livre"
	if result.FixedRegion != "" {
		regionName = result.FixedRegion.Name()
	}

	materialUnits := 0
	for _, drop := range result.Materials {
		materialUnits += drop.Quantity
	}

	equipmentUnits := 0
	for _, drop := range result.Equipment {
		equipmentUnits += drop.Quantity
	}

	fmt.Fprintf(
		&builder,
		"⚔️ *EXPEDIÇÃO* • %s • %s\n",
		regionName,
		formatRPGNumber(result.Battles),
	)

	fmt.Fprintf(
		&builder,
		"✅ %s • 💀 %s • 👑 %s/%s\n",
		formatRPGNumber(result.Wins),
		formatRPGNumber(result.Losses),
		formatRPGNumber(result.BossKills),
		formatRPGNumber(result.BossEncounters),
	)

	fmt.Fprintf(
		&builder,
		"💰 %s • 💎 %s • 📦 %s • 🎁 %s",
		formatGold(result.GoldReward),
		formatRPGNumber(result.CrystalReward),
		formatRPGNumber(materialUnits),
		formatRPGNumber(equipmentUnits),
	)

	highlights := make([]string, 0, 3)
	totalHighlighted := 0

	for _, drop := range result.Equipment {
		if drop.Item.Rarity.Rank() < rpg.RarityEpic.Rank() {
			continue
		}

		totalHighlighted++
		if len(highlights) >= 3 {
			continue
		}

		name := fmt.Sprintf(
			"%s %s",
			rpgRarityIcon(drop.Item.Rarity),
			drop.Item.Name,
		)

		if drop.Quantity > 1 {
			name += fmt.Sprintf("×%d", drop.Quantity)
		}

		highlights = append(highlights, name)
	}

	if len(highlights) > 0 {
		builder.WriteString("\n✨ ")
		builder.WriteString(strings.Join(highlights, " • "))

		if totalHighlighted > len(highlights) {
			fmt.Fprintf(
				&builder,
				" • +%d",
				totalHighlighted-len(highlights),
			)
		}
	}

	if len(autoEquipped) > 0 {
		builder.WriteString("\n⚡ ")
		builder.WriteString(strings.Join(autoEquipped, " • "))
	}

	if result.Recovery > 0 {
		fmt.Fprintf(
			&builder,
			"\n⏳ %s",
			formatPVEDuration(result.Recovery),
		)
	}

	return strings.TrimSpace(builder.String())
}
