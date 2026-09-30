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

func handleRPGDungeons(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) {

	intent,
		ok :=
		parseDungeonCommand(
			text,
		)

	if !ok {
		sendRPGDungeonUsage(
			client,
			msgEvent,
		)

		return
	}

	catalog,
		materials,
		err :=
		getRPGCatalogs()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo das dungeons",
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

	if _, err :=
		rpg.GetOrCreatePlayer(
			groupJID,
			jid,
		); err != nil {

		sendRPGPlayerError(
			client,
			msgEvent,
			err,
		)

		return
	}

	switch intent.Action {

	case dungeonActionList:

		renderRPGDungeonList(
			client,
			msgEvent,
			groupJID,
			jid,
			catalog,
			false,
		)

		return

	case dungeonActionChaosList:

		renderRPGDungeonList(
			client,
			msgEvent,
			groupJID,
			jid,
			catalog,
			true,
		)

		return

	case dungeonActionInfo:

		dungeon,
			exists :=
			rpg.DungeonByID(
				intent.Dungeon,
			)

		if !exists {
			sendRPGDungeonUsage(
				client,
				msgEvent,
			)

			return
		}

		if dungeon.Locked {
			sendRPGChaosDungeonLocked(
				client,
				msgEvent,
				dungeon,
			)

			return
		}

		renderRPGDungeonDetails(
			client,
			msgEvent,
			groupJID,
			jid,
			dungeon,
			catalog,
		)

		return

	case dungeonActionEnter:

		handleRPGDungeonAttempt(
			client,
			msgEvent,
			groupJID,
			jid,
			intent.Dungeon,
			catalog,
			materials,
		)

		return
	}

	sendRPGDungeonUsage(
		client,
		msgEvent,
	)
}

func renderRPGDungeonList(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	groupJID string,
	jid string,
	catalog *rpg.Catalog,
	chaosOnly bool,
) {
	summary, err :=
		rpg.GetEquipmentSummary(
			groupJID,
			jid,
			catalog,
		)

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"consultando Poder de Combate",
			err,
		)

		return
	}

	var builder strings.Builder
	var dungeons []rpg.Dungeon

	if chaosOnly {
		dungeons =
			rpg.ChaosDungeons()

		fmt.Fprintf(
			&builder,
			"🌑 *DUNGEONS DO CAOS*\n"+
				"━━━━━━━━━━━━━━━━━━\n"+
				"⚡ Seu Poder: *%s PC*\n\n",
			formatRPGNumber(
				summary.CombatPower,
			),
		)
	} else {
		dungeons =
			rpg.NormalDungeons()

		fmt.Fprintf(
			&builder,
			"🏰 *DUNGEONS*\n"+
				"━━━━━━━━━━━━━━━━━━\n"+
				"⚡ Seu Poder: *%s PC*\n\n",
			formatRPGNumber(
				summary.CombatPower,
			),
		)
	}

	for index, dungeon := range dungeons {

		if dungeon.Locked {
			fmt.Fprintf(
				&builder,
				"%s *%s*\n"+
					"⚡ Recomendado: %s PC\n"+
					"🔒 *SELADA*\n",
				dungeon.Emoji,
				strings.ToUpper(
					dungeon.Name,
				),
				formatRPGNumber(
					dungeon.RecommendedPower,
				),
			)
		} else {
			fmt.Fprintf(
				&builder,
				"%s *%s*\n"+
					"⚡ Recomendado: %s PC\n"+
					"🎁 Lendário: %d%%\n",
				dungeon.Emoji,
				strings.ToUpper(
					dungeon.Name,
				),
				formatRPGNumber(
					dungeon.RecommendedPower,
				),
				dungeon.EquipmentDropChance,
			)
		}

		if index <
			len(dungeons)-1 {

			builder.WriteString(
				"\n",
			)
		}
	}

	builder.WriteString(
		"\n\n━━━━━━━━━━━━━━━━━━\n",
	)

	if chaosOnly {
		builder.WriteString(
			"☄️ *O caminho permanece selado.*\n" +
				"Algo além deste mundo aguarda o Presságio...",
		)
	} else {
		builder.WriteString(
			"⚔️ Para desafiar:\n" +
				"*!dungeons <id> entrar*\n\n" +
				"🌑 *!dungeons caos*",
		)
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		builder.String(),
	)
}

func renderRPGDungeonDetails(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	groupJID string,
	jid string,
	dungeon rpg.Dungeon,
	catalog *rpg.Catalog,
) {
	summary, err :=
		rpg.GetEquipmentSummary(
			groupJID,
			jid,
			catalog,
		)

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"consultando Poder de Combate",
			err,
		)

		return
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		fmt.Sprintf(
			"%s *%s*\n"+
				"━━━━━━━━━━━━━━━━━━\n"+
				"⚡ Seu Poder: %s PC\n"+
				"⚔️ Recomendado: %s PC\n\n"+
				"💰 Gold: %s – %s\n"+
				"💎 Cristais: %d – %d\n"+
				"🟠 Lendário: %d%%\n"+
				"⏳ Recuperação: 1 min\n\n"+
				"📜 _%s_\n\n"+
				"━━━━━━━━━━━━━━━━━━\n"+
				"⚔️ *!dungeons %s entrar*",
			dungeon.Emoji,
			strings.ToUpper(
				dungeon.Name,
			),
			formatRPGNumber(
				summary.CombatPower,
			),
			formatRPGNumber(
				dungeon.RecommendedPower,
			),
			formatRPGNumber(
				dungeon.GoldMin,
			),
			formatRPGNumber(
				dungeon.GoldMax,
			),
			dungeon.CrystalMin,
			dungeon.CrystalMax,
			dungeon.EquipmentDropChance,
			dungeon.Description,
			dungeon.ID,
		),
	)

}

func handleRPGDungeonAttempt(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	groupJID string,
	jid string,
	dungeonID string,
	catalog *rpg.Catalog,
	materials *rpg.MaterialCatalog,
) {
	result, err :=
		rpg.AttemptDungeon(
			groupJID,
			jid,
			dungeonID,
			catalog,
			materials,
		)

	if err != nil {

		var cooldownErr *rpg.DungeonCooldownError

		switch {

		case errors.As(
			err,
			&cooldownErr,
		):

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"⏳ Aguarde *%s* para tentar novamente.",
					formatDungeonDuration(
						cooldownErr.Remaining,
					),
				),
			)

		case errors.Is(
			err,
			rpg.ErrDungeonLocked,
		):

			dungeon,
				_ :=
				rpg.DungeonByID(
					dungeonID,
				)

			sendRPGChaosDungeonLocked(
				client,
				msgEvent,
				dungeon,
			)

		case errors.Is(
			err,
			rpg.ErrDungeonNotFound,
		):

			sendRPGDungeonUsage(
				client,
				msgEvent,
			)

		default:

			sendRPGInternalError(
				client,
				msgEvent,
				"executando Dungeon",
				err,
			)
		}

		return
	}

	if !result.Won {

		response :=
			fmt.Sprintf(
				"💀 *DERROTA* • %s\n"+
					"🎯 %d%% • ⏳ 1 min",
				result.Dungeon.Name,
				result.SuccessChance,
			)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			response,
		)

		return
	}

	lootText := ""

	if result.Loot != nil &&
		result.LootQuantity > 0 {

		lootText =
			fmt.Sprintf(
				"\n🎒 +%d *%s*",
				result.LootQuantity,
				result.Loot.Name,
			)
	}

	autoEquipText := ""
	equipmentText := ""

	if result.EquipmentDrop != nil {
		equipmentText =
			fmt.Sprintf(
				"\n🎁 %s *%s*",
				rpgRarityIcon(
					result.EquipmentDrop.Rarity,
				),
				result.EquipmentDrop.Name,
			)

		autoEquip,
			autoEquipErr :=
			rpg.AutoEquipIfBetter(
				groupJID,
				jid,
				result.EquipmentDrop.ID,
				catalog,
			)

		if autoEquipErr == nil &&
			autoEquip != nil &&
			autoEquip.Equipped {

			autoEquipText =
				"\n⚡ *Autoequipado*"
		}
	}

	response :=
		fmt.Sprintf(
			"✅ *DUNGEON CONCLUÍDA* • %s\n"+
				"💰 +%s • 💎 +%s%s%s%s\n"+
				"⏳ 1 min",
			result.Dungeon.Name,

			formatRPGNumber(
				result.GoldReward,
			),

			formatRPGNumber(
				result.CrystalReward,
			),

			lootText,
			equipmentText,
			autoEquipText,
		)

	response +=
		recordRPGXP(
			msgEvent.Info.Chat.String(),
			canonicalSenderJID(
				msgEvent,
			),
			rpgDungeonVictoryXP(
				result,
			),
		)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}

func sendRPGDungeonUsage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"🏰 *DUNGEONS*\n\n"+
			"📜 Listar:\n"+
			"*!dungeons*\n"+
			"*!dungeons caos*\n\n"+
			"🔎 Ver detalhes:\n"+
			"*!dungeons <dungeon>*\n\n"+
			"⚔️ Entrar:\n"+
			"*!dungeons <dungeon> entrar*\n"+
			"*!dungeons entrar <dungeon>*\n"+
			"*!entrar <dungeon>*\n\n"+
			"🏰 Dungeons:\n"+
			"• *ruinas*\n"+
			"• *cripta*\n"+
			"• *fortaleza*\n\n"+
			"✨ Acentos, maiúsculas e ordem flexível são aceitos.",
	)
}

func sendRPGChaosDungeonLocked(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	dungeon rpg.Dungeon,
) {
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		fmt.Sprintf(
			"🔒 %s *%s*\n"+
				"⚡ %s PC recomendado\n"+
				"☄️ Esta Dungeon permanece selada até a próxima fase do *Presságio do Caos*.",
			dungeon.Emoji,
			dungeon.Name,
			formatRPGNumber(
				dungeon.RecommendedPower,
			),
		),
	)
}

func formatDungeonDuration(
	duration time.Duration,
) string {

	if duration <= 0 {
		return "agora"
	}

	minutes :=
		int(
			(duration +
				time.Minute -
				1) /
				time.Minute,
		)

	if minutes < 60 {
		return fmt.Sprintf(
			"%d min",
			minutes,
		)
	}

	hours :=
		minutes / 60

	remainingMinutes :=
		minutes % 60

	if remainingMinutes == 0 {
		return fmt.Sprintf(
			"%dh",
			hours,
		)
	}

	return fmt.Sprintf(
		"%dh %dmin",
		hours,
		remainingMinutes,
	)
}
