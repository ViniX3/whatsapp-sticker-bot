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

// handleUnknownCommand verifica se a mensagem parece ser
// um comando, mas não foi reconhecida pelos handlers anteriores.
//
// Exemplos:
//
// !roubo
// !roubargold
// !teste
// !qualquercoisa
//
// Mensagens normais sem "!" são ignoradas.
func handleUnknownCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts := strings.Fields(text)

	if len(parts) == 0 {
		return false
	}

	command := strings.TrimSpace(parts[0])

	if !strings.HasPrefix(command, "!") {
		return false
	}

	senderJID := msgEvent.Info.Sender.ToNonAD()

	senderName := strings.TrimSpace(
		msgEvent.Info.PushName,
	)

	if senderName == "" {
		senderName = strings.TrimSpace(
			senderJID.User,
		)
	}

	if senderName == "" {
		senderName = "usuário"
	}

	response := fmt.Sprintf(
		"@%s esse comando não existe, use !menu para verificar os comandos que existem!",
		senderName,
	)

	err := whatsapp.SendMentionedText(
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

	logger.Info(
		"Comando desconhecido recebido:",
		command,
		"de:",
		senderJID.String(),
	)

	return true
}
