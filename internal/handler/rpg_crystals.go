package handler

import (
	"fmt"

	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func handleRPGCrystals(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {

	if len(parts) != 1 {

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"💎 Use: *!cristais*",
		)

		return
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	player, err :=
		rpg.GetPlayer(
			groupJID,
			jid,
		)

	if err != nil {

		sendRPGPlayerError(
			client,
			msgEvent,
			err,
		)

		return
	}

	response :=
		fmt.Sprintf(
			"💎 *CRISTAIS MÁGICOS*\n\n"+
				"💎 Saldo atual: *%s Cristais*\n\n"+
				"⚔️ Ganhe Cristais em *!pve* e *!dungeons*.",
			formatRPGNumber(
				player.MagicCrystals,
			),
		)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}
