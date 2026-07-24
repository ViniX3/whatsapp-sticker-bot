package whatsapp

import (
	"context"
	"fmt"
	"os"
	"time"

	"whatsapp-sticker-bot/internal/logger"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
)

func DownloadImage(
	ctx context.Context,
	client *whatsmeow.Client,
	msg *waProto.ImageMessage,
) (string, error) {

	logger.Debug(
		"Iniciando download da imagem",
	)

	data, err := client.Download(
		ctx,
		msg,
	)

	if err != nil {

		logger.Error(
			"Erro no download da imagem:",
			err,
		)

		return "", err
	}

	filename := fmt.Sprintf(
		"temp/image_%d.jpg",
		time.Now().Unix(),
	)

	err = os.WriteFile(
		filename,
		data,
		0644,
	)

	if err != nil {

		logger.Error(
			"Erro salvando imagem:",
			err,
		)

		return "", err
	}

	logger.Success(
		"Imagem salva:",
		filename,
	)

	return filename, nil
}


func DownloadVideo(
	ctx context.Context,
	client *whatsmeow.Client,
	msg *waProto.VideoMessage,
) (string, error) {

	logger.Debug(
		"Iniciando download do vídeo",
	)

	data, err := client.Download(
		ctx,
		msg,
	)

	if err != nil {

		logger.Error(
			"Erro no download do vídeo:",
			err,
		)

		return "", err
	}

	filename := fmt.Sprintf(
		"temp/video_%d.mp4",
		time.Now().Unix(),
	)

	err = os.WriteFile(
		filename,
		data,
		0644,
	)

	if err != nil {

		logger.Error(
			"Erro salvando vídeo:",
			err,
		)

		return "", err
	}

	logger.Success(
		"Vídeo salvo:",
		filename,
	)

	return filename, nil
}
