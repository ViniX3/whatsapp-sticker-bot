package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"whatsapp-sticker-bot/internal/auth"
	"whatsapp-sticker-bot/internal/database"
	"whatsapp-sticker-bot/internal/handler"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	// Quantidade máxima de mensagens processadas simultaneamente.
	MessageWorkerCount = 4

	// Quantidade máxima de mensagens que podem aguardar na fila.
	MessageQueueSize = 256
)

type BotState struct {
	Ready     atomic.Bool
	StartedAt time.Time
}

func main() {

	// ==========================================================
	// BANCO DE DADOS
	// ==========================================================

	if err := database.Init(); err != nil {
		logger.Error(
			"Erro ao inicializar banco SQLite:",
			err,
		)
		return
	}

	logger.Success(
		"Banco SQLite inicializado com sucesso",
	)

	// ==========================================================
	// WHITELIST DE GRUPOS
	// ==========================================================

	if err := auth.LoadGroups("groups.json"); err != nil {
		logger.Error(
			"Erro ao carregar groups.json:",
			err,
		)
		return
	}

	logger.Success(
		"Whitelist de grupos carregada",
	)

	// ==========================================================
	// CLIENTE WHATSAPP
	// ==========================================================

	client := whatsapp.NewClient()

	state := &BotState{
		StartedAt: time.Now(),
	}

	state.Ready.Store(false)

	// ==========================================================
	// FILA INTERNA DE MENSAGENS
	// ==========================================================

	messageQueue := make(
		chan *events.Message,
		MessageQueueSize,
	)

	startMessageWorkers(
		client,
		state,
		messageQueue,
	)

	logger.Success(
		fmt.Sprintf(
			"Sistema de processamento iniciado com %d workers",
			MessageWorkerCount,
		),
	)

	// ==========================================================
	// SINCRONIZAÇÃO INICIAL
	// ==========================================================

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

	// ==========================================================
	// EVENT HANDLER DO WHATSMEOW
	// ==========================================================

	client.AddEventHandler(func(evt interface{}) {

		// Durante os primeiros segundos de inicialização,
		// eventos recebidos não são enviados para os workers.
		if !state.Ready.Load() {
			return
		}

		msg, ok := evt.(*events.Message)
		if !ok {
			return
		}

		// Evita que mensagens provenientes do History Sync
		// ocupem espaço na fila interna.
		//
		// ProcessMessage também possui essa proteção,
		// mantendo uma segunda camada de segurança.
		if msg.Info.Timestamp.Before(state.StartedAt) {
			logger.Debug(
				"Mensagem antiga ignorada antes da fila",
			)
			return
		}

		logger.Debug(
			fmt.Sprintf(
				"Mensagem recebida de %s",
				msg.Info.Sender.User,
			),
		)

		// ------------------------------------------------------
		// IMPORTANTE
		//
		// O processamento da mensagem NÃO acontece aqui.
		//
		// O callback do whatsmeow apenas coloca a mensagem
		// em uma fila interna e retorna rapidamente.
		//
		// Isso evita que operações demoradas como:
		//
		//   - ffmpeg
		//   - download de vídeo
		//   - upload de sticker
		//   - SQLite
		//   - GetGroupInfo
		//
		// bloqueiem o processamento interno do whatsmeow.
		// ------------------------------------------------------

		messageQueue <- msg

		// Alerta preventivo caso a fila comece a crescer
		// excessivamente.
		queueUsage := len(messageQueue)

		if queueUsage >= (MessageQueueSize * 80 / 100) {
			logger.Warn(
				fmt.Sprintf(
					"Fila de mensagens elevada: %d/%d",
					queueUsage,
					MessageQueueSize,
				),
			)
		}
	})

	logger.Success(
		"Bot iniciado com sistema Gold habilitado!",
	)

	select {}
}

// startMessageWorkers inicia um conjunto fixo de workers.
//
// Cada worker consome mensagens da mesma fila e chama
// ProcessMessage de forma independente.
//
// Isso permite processamento simultâneo sem criar uma
// quantidade ilimitada de goroutines.
func startMessageWorkers(
	client *whatsmeow.Client,
	state *BotState,
	messageQueue <-chan *events.Message,
) {

	for workerID := 1; workerID <= MessageWorkerCount; workerID++ {

		id := workerID

		go func() {

			logger.Debug(
				fmt.Sprintf(
					"Worker %d iniciado",
					id,
				),
			)

			for msg := range messageQueue {

				processMessageSafely(
					id,
					client,
					state,
					msg,
				)
			}
		}()
	}
}

// processMessageSafely executa o handler protegido contra panic.
//
// Um panic inesperado em uma mensagem não deve matar
// permanentemente um worker.
func processMessageSafely(
	workerID int,
	client *whatsmeow.Client,
	state *BotState,
	msg *events.Message,
) {

	defer func() {

		if recovered := recover(); recovered != nil {

			logger.Error(
				fmt.Sprintf(
					"Panic recuperado no worker %d: %v",
					workerID,
					recovered,
				),
			)
		}
	}()

	handler.ProcessMessage(
		client,
		msg,
		state.StartedAt,
	)
}
