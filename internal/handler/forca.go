package handler

import (
	"errors"
	"fmt"
	"strings"

	"whatsapp-sticker-bot/internal/forca"
	"whatsapp-sticker-bot/internal/gold"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/profile"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func handleForcaCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts :=
		strings.Fields(text)

	if len(parts) == 0 {
		return false
	}

	command :=
		strings.ToLower(
			parts[0],
		)

	switch command {

	case "!forca":
		handleForcaStart(
			client,
			msgEvent,
			parts,
		)

		return true

	case "!letra":
		handleForcaLetter(
			client,
			msgEvent,
			parts,
		)

		return true

	case "!palavra":
		handleForcaWord(
			client,
			msgEvent,
			parts,
		)

		return true
	}

	return false
}

func handleForcaStart(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {
	if !requireGoldGroup(
		client,
		msgEvent,
		"!forca",
	) {
		return
	}

	if len(parts) != 1 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"🎯 Use apenas *!forca* para iniciar uma partida.",
		)

		return
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	session, err :=
		forca.Start(
			groupJID,
		)

	if err != nil {
		if errors.Is(
			err,
			forca.ErrSessionExists,
		) {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"🎯 Já existe uma partida de Forca neste grupo!\n\n"+
					renderForcaBoard(
						session,
					),
			)

			return
		}

		logger.Error(
			"Erro iniciando !forca:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível iniciar o jogo da Forca agora.",
		)

		return
	}

	response :=
		"🎯 *JOGO DA FORCA*\n\n" +
			"Uma nova palavra foi escolhida!\n\n" +
			renderForcaBoard(
				session,
			) +
			"\n\n" +
			"Use *!letra <letra>* para tentar uma letra.\n" +
			"Use *!palavra <resposta>* para tentar resolver.\n\n" +
			"⏳ A partida expira em *10 minutos*."

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}

func handleForcaLetter(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {
	if !requireGoldGroup(
		client,
		msgEvent,
		"!letra",
	) {
		return
	}

	if len(parts) != 2 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"🔤 Uso correto: *!letra a*",
		)

		return
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	result, err :=
		forca.GuessLetter(
			groupJID,
			parts[1],
		)

	if err != nil {
		handleForcaError(
			client,
			msgEvent,
			err,
		)

		return
	}

	if result.Repeated {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"🔁 Essa letra já foi utilizada.\n\n"+
				renderForcaBoard(
					result.Session,
				),
		)

		return
	}

	if result.Won {
		finishForcaWin(
			client,
			msgEvent,
			result.Session,
		)

		return
	}

	if result.Lost {
		finishForcaLoss(
			client,
			msgEvent,
			result.Session,
		)

		return
	}

	var headline string

	if result.Correct {
		headline =
			"✅ *LETRA CORRETA!*"
	} else {
		headline =
			"❌ *ESSA LETRA NÃO EXISTE!*"
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		headline+
			"\n\n"+
			renderForcaBoard(
				result.Session,
			),
	)
}

func handleForcaWord(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {
	if !requireGoldGroup(
		client,
		msgEvent,
		"!palavra",
	) {
		return
	}

	if len(parts) < 2 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"🎯 Uso correto: *!palavra <resposta>*",
		)

		return
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	guess :=
		strings.Join(
			parts[1:],
			" ",
		)

	result, err :=
		forca.GuessWord(
			groupJID,
			guess,
		)

	if err != nil {
		handleForcaError(
			client,
			msgEvent,
			err,
		)

		return
	}

	if result.Won {
		finishForcaWin(
			client,
			msgEvent,
			result.Session,
		)

		return
	}

	if result.Lost {
		finishForcaLoss(
			client,
			msgEvent,
			result.Session,
		)

		return
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"❌ *PALAVRA INCORRETA!*\n\n"+
			renderForcaBoard(
				result.Session,
			),
	)
}

func finishForcaWin(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	session *forca.Session,
) {
	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	name :=
		strings.TrimSpace(
			msgEvent.Info.PushName,
		)

	if name == "" {
		name =
			msgEvent.Info.
				Sender.
				User
	}

	remaining :=
		forca.RemainingAttempts(
			session,
		)

	reward :=
		1000

	if remaining > 1 {
		reward +=
			(remaining - 1) * 120
	}

	balance, goldErr :=
		gold.RewardForca(
			groupJID,
			jid,
			reward,
		)

	if goldErr != nil {
		logger.Error(
			"Erro creditando Gold do Forca:",
			goldErr,
		)
	}

	progressText := ""

	xpResult, xpErr :=
		profile.RecordForcaWin(
			groupJID,
			jid,
		)

	if xpErr != nil {
		logger.Error(
			"Erro registrando XP do Forca:",
			xpErr,
		)
	} else {
		progressText =
			progressionSuffix(
				xpResult,
			)
	}

	response := fmt.Sprintf(
		"@%s\n\n"+
			"🏆 *VOCÊ VENCEU A FORCA!*\n\n"+
			"🎯 Palavra: *%s*\n"+
			"📚 Categoria: *%s*\n\n",
		name,
		session.Word,
		session.Category,
	)

	if goldErr == nil {
		response += fmt.Sprintf(
			"💰 Recompensa: *+%d Gold*\n"+
				"💰 Saldo: *%d Gold*",
			reward,
			balance,
		)
	} else {
		response +=
			"⚠️ Não foi possível creditar a recompensa em Gold."
	}

	response +=
		progressText

	err :=
		whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{
				msgEvent.Info.
					Sender.
					ToNonAD(),
			},
		)

	if err != nil {
		logger.Error(
			"Erro enviando vitória do Forca:",
			err,
		)
	}
}

func finishForcaLoss(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	session *forca.Session,
) {
	response := fmt.Sprintf(
		"💀 *FIM DE JOGO!*\n\n"+
			"%s\n\n"+
			"Vocês não conseguiram descobrir a palavra.\n\n"+
			"🎯 Resposta: *%s*\n"+
			"📚 Categoria: *%s*\n\n"+
			"Use *!forca* para começar outra partida.",
		hangmanArt(
			forca.MaxErrors,
		),
		session.Word,
		session.Category,
	)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}

func handleForcaError(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	err error,
) {
	var response string

	switch {

	case errors.Is(
		err,
		forca.ErrSessionMissing,
	):
		response =
			"🎯 Não existe nenhuma partida de Forca ativa.\n\nUse *!forca* para começar."

	case errors.Is(
		err,
		forca.ErrSessionExpired,
	):
		response =
			"⌛ A partida de Forca expirou.\n\nUse *!forca* para começar outra."

	case errors.Is(
		err,
		forca.ErrInvalidLetter,
	):
		response =
			"🔤 Informe apenas uma letra.\n\nExemplo: *!letra a*"

	case errors.Is(
		err,
		forca.ErrInvalidWord,
	):
		response =
			"🎯 Informe uma palavra válida."

	default:
		logger.Error(
			"Erro no jogo da Forca:",
			err,
		)

		response =
			"❌ Não foi possível processar o jogo da Forca agora."
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}

func renderForcaBoard(
	session *forca.Session,
) string {
	if session == nil {
		return ""
	}

	wrong :=
		"nenhuma"

	if len(session.Wrong) > 0 {
		values :=
			make(
				[]string,
				0,
				len(session.Wrong),
			)

		for _, letter := range session.Wrong {

			values =
				append(
					values,
					strings.ToUpper(
						string(letter),
					),
				)
		}

		wrong =
			strings.Join(
				values,
				", ",
			)
	}

	return fmt.Sprintf(
		"%s\n\n"+
			"📚 Categoria: *%s*\n\n"+
			"🔤 %s\n\n"+
			"❌ Erros: *%d/%d*\n"+
			"🔴 Letras erradas: *%s*\n"+
			"❤️ Tentativas restantes: *%d*",
		hangmanArt(
			session.Errors,
		),
		session.Category,
		forca.Mask(
			session,
		),
		session.Errors,
		forca.MaxErrors,
		wrong,
		forca.RemainingAttempts(
			session,
		),
	)
}

func hangmanArt(
	errors int,
) string {
	stages := []string{
		"┌─────┐\n│\n│\n│\n│\n┴──────",
		"┌─────┐\n│     😵\n│\n│\n│\n┴──────",
		"┌─────┐\n│     😵\n│      │\n│\n│\n┴──────",
		"┌─────┐\n│     😵\n│     /│\n│\n│\n┴──────",
		"┌─────┐\n│     😵\n│     /│\\\n│\n│\n┴──────",
		"┌─────┐\n│     😵\n│     /│\\\n│     /\n│\n┴──────",
		"┌─────┐\n│     💀\n│     /│\\\n│     / \\\n│\n┴──────",
	}

	if errors < 0 {
		errors = 0
	}

	if errors >= len(stages) {
		errors =
			len(stages) - 1
	}

	return stages[errors]
}
