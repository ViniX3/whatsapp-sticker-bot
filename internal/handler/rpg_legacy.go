package handler

import (
	"fmt"

	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func handleLegacyRPGEquipmentCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	command string,
) {
	if !requireGoldGroup(
		client,
		msgEvent,
		command,
	) {
		return
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	catalog, _, err :=
		getRPGCatalogs()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo RPG",
			err,
		)

		return
	}

	claimed, err :=
		rpg.HasClaimedStarterSet(
			groupJID,
			jid,
		)

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"consultando personagem RPG",
			err,
		)

		return
	}

	// Usuário antigo ou novo ainda sem RPG:
	// o comando legado passa a funcionar como
	// uma porta para o onboarding unificado.
	if !claimed {
		handleAdventureOnboarding(
			client,
			msgEvent,
			"!rpg",
		)

		return
	}

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
			"consultando equipamentos RPG",
			err,
		)

		return
	}

	var equipmentName string

	switch command {
	case "!espada":
		equipmentName = "arma"

	case "!escudo":
		equipmentName = "escudo"

	case "!armadura":
		equipmentName = "armadura"

	default:
		equipmentName = "equipamento"
	}

	response :=
		fmt.Sprintf(
			"╔════════════════════╗\n"+
				"⚔️ *SISTEMA DE EQUIPAMENTOS*\n"+
				"╚════════════════════╝\n\n"+
				"📜 O comando *%s* agora faz parte do sistema RPG.\n\n"+
				"Seu %s não é mais uma proteção temporária nem uma compra separada.\n"+
				"Ele faz parte do seu arsenal e contribui diretamente para seu Poder de Combate.\n\n"+
				"🏆 Poder de Combate atual: *%s*\n\n"+
				"🛡️ Veja o que está equipado:\n"+
				"*!equipamentos*\n\n"+
				"🎒 Veja seus itens disponíveis:\n"+
				"*!inventario*\n\n"+
				"⚒️ Para equipar outro item:\n"+
				"*!equipar <id>*",
			command,
			equipmentName,
			formatRPGNumber(
				summary.CombatPower,
			),
		)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}
