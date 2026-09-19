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

func handleSlotsCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts := strings.Fields(text)

	if len(parts) == 0 ||
		!strings.EqualFold(
			parts[0],
			"!slots",
		) {
		return false
	}

	if !requireGoldGroup(
		client,
		msgEvent,
		"!slots",
	) {
		return true
	}

	if len(parts) != 2 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"🎰 Uso correto: *!slots <valor>*\n\nExemplo: *!slots 500*\n\n💰 Aposta mínima: *%d Gold*",
				gold.SlotsMinBet,
			),
		)

		return true
	}

	amount, err := strconv.Atoi(
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

	if amount < gold.SlotsMinBet {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"🎰 A aposta mínima do *!slots* é de *%d Gold*.",
				gold.SlotsMinBet,
			),
		)

		return true
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	name := strings.TrimSpace(
		msgEvent.Info.PushName,
	)

	if name == "" {
		name = msgEvent.Info.Sender.User
	}

	result, err := gold.PlaySlots(
		groupJID,
		jid,
		amount,
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
				"❌ Você não possui Gold suficiente para essa rodada.",
			)

		case errors.Is(
			err,
			gold.ErrSlotsInvalidBet,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"🎰 A aposta mínima do *!slots* é de *%d Gold*.",
					gold.SlotsMinBet,
				),
			)

		default:
			logger.Error(
				"Erro ao executar !slots:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível jogar no Slots agora.",
			)
		}

		return true
	}

	progressText :=
		recordSlotsProgress(
			groupJID,
			jid,
			result.Multiplier,
			result.Prize,
		)

	machine := fmt.Sprintf(
		"┃ %s ┃ %s ┃ %s ┃",
		result.Symbols[0],
		result.Symbols[1],
		result.Symbols[2],
	)

	var response string

	switch result.Multiplier {
	case 0:
		response = fmt.Sprintf(
			"🎰 *SLOT MACHINE* 🎰\n\n%s\n\n💀 *NÃO FOI DESSA VEZ!*\n\n💰 Aposta: *%d Gold*\n💸 Perda: *-%d Gold*\n💰 Saldo: *%d Gold*",
			machine,
			result.BetAmount,
			result.BetAmount,
			result.Balance,
		)

	case 1:
		response = fmt.Sprintf(
			"🎰 *SLOT MACHINE* 🎰\n\n%s\n\n🍒 *RECUPEROU!*\n\n💰 Aposta: *%d Gold*\n↩️ Retorno: *%d Gold*\n💰 Saldo: *%d Gold*",
			machine,
			result.BetAmount,
			result.Prize,
			result.Balance,
		)

	default:
		response = fmt.Sprintf(
			"🎰 *SLOT MACHINE* 🎰\n\n%s\n\n🔥 *%s!*\n\n💰 Aposta: *%d Gold*\n🏆 Prêmio: *%d Gold*\n📈 Lucro: *+%d Gold*\n✖️ Multiplicador: *%dx*\n💰 Saldo: *%d Gold*",
			machine,
			result.Tier,
			result.BetAmount,
			result.Prize,
			result.NetResult,
			result.Multiplier,
			result.Balance,
		)
	}

	response = fmt.Sprintf(
		"@%s\n\n%s%s",
		name,
		response,
		progressText,
	)

	err = whatsapp.SendMentionedText(
		client,
		msgEvent.Info.Chat,
		response,
		[]types.JID{
			msgEvent.Info.Sender.ToNonAD(),
		},
	)

	if err != nil {
		logger.Error(
			"Erro ao enviar resultado do !slots:",
			err,
		)
	}

	logger.Info(
		"Slots executado:",
		"Usuário:",
		jid,
		"Grupo:",
		groupJID,
		"Aposta:",
		result.BetAmount,
		"Multiplicador:",
		result.Multiplier,
		"Prêmio:",
		result.Prize,
	)

	return true
}

func recordSlotsProgress(
	groupJID string,
	jid string,
	multiplier int,
	prize int,
) string {
	result, err := profile.RecordSlotResult(
		groupJID,
		jid,
		multiplier,
		prize,
	)

	if err != nil {
		logger.Error(
			"Erro ao registrar progresso do Slots:",
			err,
		)
		return ""
	}

	return progressionSuffix(result)
}
