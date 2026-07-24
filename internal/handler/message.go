package handler

import (
	"strings"
	"time"

	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/media"
	"whatsapp-sticker-bot/internal/processor"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func ProcessMessage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	botStartedAt time.Time,
) {

	// Ignora mensagens antigas
	if msgEvent.Info.Timestamp.Before(botStartedAt) {

		logger.Debug(
			"Mensagem antiga ignorada",
		)

		return
	}


	logger.Info("==============================")
	logger.Info("Mensagem recebida")


	text := extractText(msgEvent)

	logger.Info(
		"Texto recebido:",
		text,
	)


	if !IsStickerCommand(text) {

		logger.Debug(
			"Sem comando !f, ignorando",
		)

		logger.Info("==============================")

		return
	}


	logger.Info(
		"Comando !f recebido",
	)


	mediaMessage := getMedia(msgEvent)


	if mediaMessage == nil {

		logger.Warn(
			"Nenhuma mídia encontrada",
		)

		logger.Info("==============================")

		return
	}


	processStickerCommand(
		client,
		msgEvent,
		mediaMessage,
	)


	logger.Info("==============================")
}


func extractText(
	msg *events.Message,
) string {


	if msg.Message.GetConversation() != "" {

		return strings.TrimSpace(
			msg.Message.GetConversation(),
		)
	}


	if msg.Message.ExtendedTextMessage != nil {

		return strings.TrimSpace(
			msg.Message.GetExtendedTextMessage().GetText(),
		)
	}


	if msg.Message.ImageMessage != nil {

		return strings.TrimSpace(
			msg.Message.GetImageMessage().GetCaption(),
		)
	}


	if msg.Message.VideoMessage != nil {

		return strings.TrimSpace(
			msg.Message.GetVideoMessage().GetCaption(),
		)
	}


	return ""
}


func getMedia(
	msg *events.Message,
) *media.Media {


	// Imagem enviada diretamente
	if msg.Message.ImageMessage != nil {

		return &media.Media{
			Type: media.Image,
			Image: msg.Message.ImageMessage,
		}
	}


	// Vídeo enviado diretamente
	if msg.Message.VideoMessage != nil {

		return &media.Media{
			Type: media.Video,
			Video: msg.Message.VideoMessage,
		}
	}


	// Mensagem respondida
	if msg.Message.ExtendedTextMessage != nil {

		contextInfo :=
			msg.Message.
				ExtendedTextMessage.
				ContextInfo


		if contextInfo != nil &&
			contextInfo.QuotedMessage != nil {


			// Imagem respondida
			if contextInfo.QuotedMessage.ImageMessage != nil {

				return &media.Media{
					Type: media.Image,
					Image: contextInfo.QuotedMessage.ImageMessage,
				}
			}


			// Vídeo respondido
			if contextInfo.QuotedMessage.VideoMessage != nil {

				return &media.Media{
					Type: media.Video,
					Video: contextInfo.QuotedMessage.VideoMessage,
				}
			}
		}
	}


	return nil
}


func processStickerCommand(
	client *whatsmeow.Client,
	msg *events.Message,
	mediaMessage *media.Media,
) {


	err := processor.ProcessSticker(
		client,
		msg.Info.Chat,
		mediaMessage,
	)


	if err != nil {

		logger.Error(
			"Erro ao processar figurinha:",
			err,
		)

		return
	}


	logger.Success(
		"Processamento concluído",
	)
}
