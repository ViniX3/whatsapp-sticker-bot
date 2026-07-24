package processor

import (
	"context"
	"fmt"
	"os"

	"whatsapp-sticker-bot/internal/converter"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/media"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func ProcessSticker(
	client *whatsmeow.Client,
	jid types.JID,
	mediaMessage *media.Media,
) error {

	logger.Info(
		"Iniciando processamento da figurinha",
	)

	var (
		mediaPath string
		err       error
	)

	switch mediaMessage.Type {

	case media.Image:

		logger.Info("Baixando imagem...")

		mediaPath, err = whatsapp.DownloadImage(
			context.Background(),
			client,
			mediaMessage.Image,
		)

	case media.Video:

		mediaPath, err = whatsapp.DownloadVideo(
		context.Background(),
		client,
		mediaMessage.Video,
	)

	default:

		return fmt.Errorf("tipo de mídia inválido")
	}

	if err != nil {

		logger.Error(
			"Erro ao baixar mídia:",
			err,
		)

		return fmt.Errorf(
			"erro ao baixar mídia: %w",
			err,
		)
	}

	logger.Info(
		"Mídia baixada:",
		mediaPath,
	)

	stickerPath, err := converter.CreateSticker(
		mediaPath,
	)

	if err != nil {

		logger.Error(
			"Erro ao criar figurinha:",
			err,
		)

		_ = os.Remove(mediaPath)

		return fmt.Errorf(
			"erro ao criar figurinha: %w",
			err,
		)
	}

	logger.Success(
		"Sticker criado:",
		stickerPath,
	)

	err = whatsapp.SendSticker(
		client,
		jid,
		stickerPath,
	)

	if err != nil {

		logger.Error(
			"Erro ao enviar figurinha:",
			err,
		)

		_ = os.Remove(mediaPath)
		_ = os.Remove(stickerPath)

		return fmt.Errorf(
			"erro ao enviar figurinha: %w",
			err,
		)
	}

	logger.Success(
		"Sticker enviado com sucesso",
	)

	cleanupFiles(
		mediaPath,
		stickerPath,
	)

	logger.Info(
		"Processamento finalizado",
	)

	return nil
}

func cleanupFiles(
	files ...string,
) {

	for _, file := range files {

		err := os.Remove(file)

		if err != nil {

			logger.Warn(
				"Não foi possível remover:",
				file,
			)

			continue
		}

		logger.Debug(
			"Arquivo removido:",
			file,
		)
	}
}
