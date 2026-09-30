package handler

import (
	"errors"

	"whatsapp-sticker-bot/internal/gold"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// resolveFlexibleGameGoldAmount transforma um argumento de
// aposta em um valor absoluto de Gold.
//
// Exemplos:
//
//	100000
//	100.000
//	all
//	tudo
//	metade
//	20%
//
// Valores relativos são sempre calculados sobre o saldo
// atual do próprio jogador.
func resolveFlexibleGameGoldAmount(
	msgEvent *events.Message,
	raw string,
) (
	int,
	error,
) {

	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	balance,
		err :=
		gold.GetBalance(
			groupJID,
			jid,
		)

	if err != nil {
		return 0,
			err
	}

	return parseFlexibleGoldAmount(
		raw,
		balance,
	)
}

func sendFlexibleGameGoldAmountError(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	err error,
) {

	message :=
		"❌ Valor de aposta inválido.\n\n" +
			"Você pode usar:\n" +
			"• *100000*\n" +
			"• *100.000*\n" +
			"• *20%*\n" +
			"• *metade*\n" +
			"• *all*\n" +
			"• *tudo*"

	switch {

	case errors.Is(
		err,
		gold.ErrWalletNotFound,
	):
		message =
			"❌ Você ainda não possui uma carteira Gold neste grupo.\n\n" +
				"Use *!gold* primeiro."

	case errors.Is(
		err,
		errFlexibleAmountPercentRange,
	):
		message =
			"❌ O percentual precisa estar entre *1% e 100%*.\n\n" +
				"Exemplo: *20%*"

	case errors.Is(
		err,
		errFlexibleAmountNoBalance,
	):
		message =
			"❌ Você não possui Gold disponível para essa aposta.\n\n" +
				"Use *!saldo* para consultar sua carteira."

	case errors.Is(
		err,
		errFlexibleAmountInvalid,
	):
		// Mantém a mensagem padrão.

	default:
		logger.Error(
			"Erro resolvendo valor flexível de jogo:",
			err,
		)

		message =
			"❌ Não foi possível consultar seu saldo Gold agora."
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		message,
	)
}
