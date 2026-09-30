package handler

import (
	"fmt"
	"strings"

	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// handleUnknownCommand verifica mensagens que parecem comandos,
// mas não foram reconhecidas pelos handlers anteriores.
//
// IMPORTANTE:
//
// Uma sugestão nunca é executada automaticamente.
// O usuário precisa enviar o comando correto.
//
// Isso evita ações acidentais principalmente em comandos como:
//
// !bet
// !pix
// !comprar
// !forjar
func handleUnknownCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {

	parts :=
		strings.Fields(
			text,
		)

	if len(parts) == 0 {
		return false
	}

	command :=
		strings.TrimSpace(
			parts[0],
		)

	if !strings.HasPrefix(
		command,
		"!",
	) {
		return false
	}

	senderJID :=
		msgEvent.Info.Sender.ToNonAD()

	senderName :=
		strings.TrimSpace(
			msgEvent.Info.PushName,
		)

	if senderName == "" {
		senderName =
			strings.TrimSpace(
				senderJID.User,
			)
	}

	if senderName == "" {
		senderName =
			"usuário"
	}

	normalizedCommand :=
		normalizeCommandToken(
			command,
		)

	suggestion,
		hasSuggestion :=
		suggestCommand(
			command,
		)

	var response string

	if hasSuggestion {

		response =
			fmt.Sprintf(
				"@%s não encontrei o comando *%s*.\n\n"+
					"💡 Você quis dizer *%s*?\n\n"+
					"📖 Use *!menu* para consultar todos os comandos.",
				senderName,
				command,
				suggestion,
			)

	} else {

		response =
			fmt.Sprintf(
				"@%s não encontrei o comando *%s*.\n\n"+
					"📖 Use *!menu* para consultar os comandos disponíveis.",
				senderName,
				command,
			)
	}

	err :=
		whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{
				senderJID,
			},
		)

	if err != nil {
		logger.Error(
			"Erro ao enviar aviso de comando desconhecido:",
			err,
		)

		return true
	}

	if hasSuggestion {

		logger.Info(
			"UNKNOWN_COMMAND:",
			"original=",
			command,
			"normalized=",
			normalizedCommand,
			"suggestion=",
			suggestion,
			"user=",
			senderJID.String(),
		)

	} else {

		logger.Info(
			"UNKNOWN_COMMAND:",
			"original=",
			command,
			"normalized=",
			normalizedCommand,
			"suggestion=none",
			"user=",
			senderJID.String(),
		)
	}

	return true
}
