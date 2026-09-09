package whatsapp

import (
	"bytes"
	"fmt"

	"github.com/agusira/webpexif"
)

func AddStickerMetadata(data []byte) ([]byte, error) {

	const (
		packID    = "wall-e-whatsapp-sticker-bot"
		packName  = "🤖 WALL-E • WhatsApp Sticker Bot"
		publisher = "✨ Figurinha criada automaticamente • 👤 Vinicius Barbosa"
	)

	riff, err := webpexif.ReadRIFF(
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"erro lendo WebP: %w",
			err,
		)
	}

	meta := webpexif.StickerMetadata{
		PackID:    packID,
		PackName:  packName,
		Publisher: publisher,
		Emojis: []string{
			"🤖",
			"✨",
		},
	}

	exif, err := webpexif.BuildExif(meta)
	if err != nil {
		return nil, fmt.Errorf(
			"erro criando metadata EXIF: %w",
			err,
		)
	}

	if err := riff.AddExif(exif); err != nil {
		return nil, fmt.Errorf(
			"erro adicionando metadata EXIF: %w",
			err,
		)
	}

	result, err := riff.Byte()
	if err != nil {
		return nil, fmt.Errorf(
			"erro reconstruindo WebP: %w",
			err,
		)
	}

	return result, nil
}
