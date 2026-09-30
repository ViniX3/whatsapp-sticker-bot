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

	bonus :=
		rpg.DungeonSuccessBonus(
			summary,
		)

	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"🏰 *DUNGEONS* • ⚡ %s PC\n\n",
		formatRPGNumber(
			summary.CombatPower,
		),
	)

	for _, dungeon := range rpg.Dungeons() {

		chance :=
			rpg.DungeonSuccessChance(
				summary.CombatPower,
				dungeon.RecommendedPower,
				bonus,
			)

		fmt.Fprintf(
			&builder,
			"%s *%s* • ⚡%s • 🎯%d%%\n",
			dungeon.Emoji,
			dungeon.ID,
			formatRPGNumber(
				dungeon.RecommendedPower,
			),
			chance,
		)
	}

	builder.WriteString(
		"\n⚔️ *!dungeons <id> entrar*",
	)

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

	bonus :=
		rpg.DungeonSuccessBonus(
			summary,
		)

	chance :=
		rpg.DungeonSuccessChance(
			summary.CombatPower,
			dungeon.RecommendedPower,
			bonus,
		)

	response :=
		fmt.Sprintf(
			"%s *%s*\n"+
				"⚡ %s/%s PC • 🎯%d%%\n"+
				"💰 %s–%s • 💎 %d–%d\n"+
				"⏳ 1 min\n\n"+
				"⚔️ *!dungeons %s entrar*",
			dungeon.Emoji,
			dungeon.Name,

			formatRPGNumber(
				summary.CombatPower,
			),

			formatRPGNumber(
				dungeon.RecommendedPower,
			),

			chance,

			formatRPGNumber(
				dungeon.GoldMin,
			),

			formatRPGNumber(
				dungeon.GoldMax,
			),

			dungeon.CrystalMin,
			dungeon.CrystalMax,

			dungeon.ID,
		)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
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

	response :=
		fmt.Sprintf(
			"✅ *DUNGEON CONCLUÍDA* • %s\n"+
				"💰 +%s • 💎 +%s%s\n"+
				"⏳ 1 min",
			result.Dungeon.Name,

			formatRPGNumber(
				result.GoldReward,
			),

			formatRPGNumber(
				result.CrystalReward,
			),

			lootText,
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
			"*!dungeons*\n\n"+
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
