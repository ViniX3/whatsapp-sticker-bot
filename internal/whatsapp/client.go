package whatsapp

import (
	"context"
	"os"

	"whatsapp-sticker-bot/internal/logger"

	"github.com/mdp/qrterminal/v3"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
)

func NewClient() *whatsmeow.Client {

	ctx := context.Background()

	waLogger := waLog.Stdout(
		"WhatsApp",
		"INFO",
		true,
	)

	container, err := sqlstore.New(
		ctx,
		"sqlite3",
		"file:storage/sessions/session.db?_foreign_keys=on",
		waLogger,
	)

	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)

	if err != nil {
		panic(err)
	}

	client := whatsmeow.NewClient(
		deviceStore,
		waLogger,
	)

	if client.Store.ID == nil {

		logger.Info(
			"Aguardando QR Code...",
		)

		qrChan, err := client.GetQRChannel(ctx)

		if err != nil {
			panic(err)
		}

		err = client.Connect()

		if err != nil {
			panic(err)
		}

		for evt := range qrChan {

			if evt.Event == "code" {

				qrterminal.GenerateHalfBlock(
					evt.Code,
					qrterminal.L,
					os.Stdout,
				)

			} else {

				logger.Info(
					"QR Status:",
					evt.Event,
				)

			}
		}

	} else {

		logger.Info(
			"Sessão encontrada. Conectando...",
		)

		err := client.Connect()

		if err != nil {
			panic(err)
		}
	}

	logger.Success(
		"WhatsApp conectado!",
	)

	return client
}
