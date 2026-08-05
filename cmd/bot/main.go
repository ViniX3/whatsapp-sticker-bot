package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"whatsapp-sticker-bot/internal/auth"
	"whatsapp-sticker-bot/internal/handler"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow/types/events"
)

type BotState struct {
	Ready     atomic.Bool
	StartedAt time.Time
}

func main() {

	// Carrega a whitelist de grupos
	if err := auth.LoadGroups("groups.json"); err != nil {
		logger.Error("Erro ao carregar groups.json:", err)
		return
	}

	client := whatsapp.NewClient()

	state := &BotState{
		StartedAt: time.Now(),
	}

	state.Ready.Store(false)

	go func() {

		logger.Info(
			"Aguardando sincronização inicial do WhatsApp...",
		)

		time.Sleep(5 * time.Second)

		state.Ready.Store(true)

		logger.Success(
			"Sincronização concluída. Bot pronto.",
		)
	}()

	client.AddEventHandler(func(evt interface{}) {

		if !state.Ready.Load() {
			return
		}

		msg, ok := evt.(*events.Message)
		if !ok {
			return
		}

		logger.Debug( 
			fmt.Sprintf( 
				"Mensagem recebida de %s",
				msg.Info.Sender.User,
			),
		)

		handler.ProcessMessage(
			client,
			msg,
			state.StartedAt,
		)
	})

	logger.Success("Bot iniciado!")

	select {}
}
