package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"

	"google.golang.org/protobuf/proto"
)

var ErrInvalidImageFile = errors.New(
	"arquivo de imagem inválido",
)

func SendImageFileWithCaption(
	client *whatsmeow.Client,
	jid types.JID,
	imagePath string,
	caption string,
) error {
	if client == nil {
		return errors.New(
			"cliente WhatsApp não inicializado",
		)
	}

	imagePath =
		strings.TrimSpace(
			imagePath,
		)

	if imagePath == "" {
		return ErrInvalidImageFile
	}

	data, err :=
		os.ReadFile(
			imagePath,
		)

	if err != nil {
		return fmt.Errorf(
			"erro lendo imagem %s: %w",
			imagePath,
			err,
		)
	}

	if len(data) == 0 {
		return fmt.Errorf(
			"%w: arquivo vazio",
			ErrInvalidImageFile,
		)
	}

	mimeType :=
		http.DetectContentType(
			data,
		)

	if !strings.HasPrefix(
		mimeType,
		"image/",
	) {
		return fmt.Errorf(
			"%w: MIME detectado %s",
			ErrInvalidImageFile,
			mimeType,
		)
	}

	uploaded, err :=
		client.Upload(
			context.Background(),
			data,
			whatsmeow.MediaImage,
		)

	if err != nil {
		return fmt.Errorf(
			"erro enviando imagem ao WhatsApp: %w",
			err,
		)
	}

	message :=
		&waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption: proto.String(
					caption,
				),

				Mimetype: proto.String(
					mimeType,
				),

				URL: &uploaded.URL,

				DirectPath: &uploaded.DirectPath,

				MediaKey: uploaded.MediaKey,

				FileEncSHA256: uploaded.FileEncSHA256,

				FileSHA256: uploaded.FileSHA256,

				FileLength: &uploaded.FileLength,
			},
		}

	_, err =
		client.SendMessage(
			context.Background(),
			jid,
			message,
		)

	if err != nil {
		return fmt.Errorf(
			"erro enviando ImageMessage: %w",
			err,
		)
	}

	return nil
}
