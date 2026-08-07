package whatsapp

import (
	"context"
	"fmt"
	"os"

	"whatsapp-sticker-bot/internal/logger"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"

	"google.golang.org/protobuf/proto"
)

func SendSticker(
	client *whatsmeow.Client,
	jid types.JID,
	stickerPath string,
) error {

	data, err := os.ReadFile(stickerPath)

	if err != nil {
		return fmt.Errorf(
			"erro lendo sticker: %w",
			err,
		)
	}

	logger.Debug(
		"Arquivo sticker carregado:",
		stickerPath,
	)

	uploaded, err := client.Upload(
		context.Background(),
		data,
		whatsmeow.MediaImage,
	)

	if err != nil {
		return fmt.Errorf(
			"erro upload sticker: %w",
			err,
		)
	}

	logger.Info(
		"Upload concluído",
	)

	logger.Debug(
		"URL:",
		uploaded.URL,
	)

	logger.Debug(
		"DirectPath:",
		uploaded.DirectPath,
	)

	msg := &waProto.Message{
		StickerMessage: &waProto.StickerMessage{

			URL: proto.String(
				uploaded.URL,
			),

			DirectPath: proto.String(
				uploaded.DirectPath,
			),

			MediaKey: uploaded.MediaKey,

			FileEncSHA256: uploaded.FileEncSHA256,

			FileSHA256: uploaded.FileSHA256,

			FileLength: proto.Uint64(
				uint64(len(data)),
			),

			Mimetype: proto.String(
				"image/webp",
			),

			IsAnimated: proto.Bool(
				false,
			),
		},
	}

	resp, err := client.SendMessage(
		context.Background(),
		jid,
		msg,
	)

	if err != nil {
		return fmt.Errorf(
			"erro enviando sticker: %w",
			err,
		)
	}

	logger.Debug(
		"Resposta SendMessage:",
		resp,
	)

	logger.Success(
		"Sticker enviado com sucesso",
	)

	return nil
}

func SendText(
        client *whatsmeow.Client,
        chat types.JID,
        text string,
) error {

        _, err := client.SendMessage(
                context.Background(),
                chat,
                &waProto.Message{
                        Conversation: &text,
                },
        )

        return err
}
