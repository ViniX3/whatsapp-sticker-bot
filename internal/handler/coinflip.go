package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"whatsapp-sticker-bot/internal/gold"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/profile"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func handleCoinflipCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts := strings.Fields(text)

	if len(parts) == 0 ||
		!strings.EqualFold(
			parts[0],
			"!caraoucoroa",
		) {

		return false
	}

	if !requireGoldGroup(
		client,
		msgEvent,
		"!caraoucoroa",
	) {
		return true
	}

	if len(parts) != 3 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"🪙 *CARA OU COROA*\n\n"+
					"Uso correto:\n"+
					"*!caraoucoroa <valor> <cara|coroa>*\n\n"+
					"Exemplo:\n"+
					"*!caraoucoroa 1000 cara*\n\n"+
					"💰 Aposta mínima: *%d Gold*",
				gold.CoinflipMinBet,
			),
		)

		return true
	}

	amount, err :=
		strconv.Atoi(
			parts[1],
		)

	if err != nil {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ O valor da aposta precisa ser um número inteiro.",
		)

		return true
	}

	if amount < gold.CoinflipMinBet {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"🪙 A aposta mínima do *!caraoucoroa* é de *%d Gold*.",
				gold.CoinflipMinBet,
			),
		)

		return true
	}

	choice :=
		strings.ToLower(
			strings.TrimSpace(
				parts[2],
			),
		)

	if choice != gold.CoinflipHeads &&
		choice != gold.CoinflipTails {

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Escolha *cara* ou *coroa*.\n\nExemplo: *!caraoucoroa 1000 cara*",
		)

		return true
	}

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

	result, err :=
		gold.PlayCoinflip(
			groupJID,
			jid,
			amount,
			choice,
		)

	if err != nil {
		switch {

		case errors.Is(
			err,
			gold.ErrWalletNotFound,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"❌ Você ainda não possui uma carteira Gold neste grupo.\n\nUse *!gold* para receber seus *%d Gold* iniciais.",
					gold.InitialGold,
				),
			)

		case errors.Is(
			err,
			gold.ErrInsufficientGold,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Você não possui Gold suficiente para essa aposta.",
			)

		case errors.Is(
			err,
			gold.ErrCoinflipInvalidBet,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"❌ A aposta mínima é de *%d Gold*.",
					gold.CoinflipMinBet,
				),
			)

		case errors.Is(
			err,
			gold.ErrCoinflipInvalidChoice,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Escolha *cara* ou *coroa*.",
			)

		default:
			logger.Error(
				"Erro ao executar !caraoucoroa:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível lançar a moeda agora.",
			)
		}

		return true
	}

	if err :=
		profile.RecordCoinflipResult(
			groupJID,
			jid,
			result.Won,
		); err != nil {

		logger.Error(
			"Erro ao registrar estatística do Cara ou Coroa:",
			err,
		)
	}

	choiceText :=
		strings.ToUpper(
			result.Choice,
		)

	outcomeText :=
		strings.ToUpper(
			result.Outcome,
		)

	var response string

	if result.Won {
		response = fmt.Sprintf(
			"@%s\n\n"+
				"🪙 *CARA OU COROA*\n\n"+
				"🎯 Você escolheu: *%s*\n"+
				"💰 Aposta: *%d Gold*\n\n"+
				"🪙 A moeda foi lançada...\n\n"+
				"*%s!*\n\n"+
				"🎉 *VOCÊ VENCEU!*\n\n"+
				"🏆 Retorno: *%d Gold*\n"+
				"📈 Lucro: *+%d Gold*\n"+
				"💰 Saldo: *%d Gold*",
			name,
			choiceText,
			result.BetAmount,
			outcomeText,
			result.Prize,
			result.NetResult,
			result.Balance,
		)
	} else {
		response = fmt.Sprintf(
			"@%s\n\n"+
				"🪙 *CARA OU COROA*\n\n"+
				"🎯 Você escolheu: *%s*\n"+
				"💰 Aposta: *%d Gold*\n\n"+
				"🪙 A moeda foi lançada...\n\n"+
				"*%s!*\n\n"+
				"💀 *VOCÊ PERDEU!*\n\n"+
				"📉 Perda: *-%d Gold*\n"+
				"💰 Saldo: *%d Gold*",
			name,
			choiceText,
			result.BetAmount,
			outcomeText,
			result.BetAmount,
			result.Balance,
		)
	}

	err =
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
			"Erro ao enviar resultado do !caraoucoroa:",
			err,
		)

		return true
	}

	logger.Info(
		"Cara ou Coroa executado:",
		"Usuário:",
		jid,
		"Grupo:",
		groupJID,
		"Aposta:",
		result.BetAmount,
		"Escolha:",
		result.Choice,
		"Resultado:",
		result.Outcome,
		"Vitória:",
		result.Won,
	)

	return true
}
