package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"whatsapp-sticker-bot/internal/gold"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/lottery"
	"whatsapp-sticker-bot/internal/profile"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func handleLotteryCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts :=
		strings.Fields(text)

	if len(parts) == 0 ||
		!strings.EqualFold(
			parts[0],
			"!loteria",
		) {

		return false
	}

	if !requireGoldGroup(
		client,
		msgEvent,
		"!loteria",
	) {
		return true
	}

	switch len(parts) {

	case 1:
		handleLotteryStatus(
			client,
			msgEvent,
		)

	case 2:
		handleLotteryPurchase(
			client,
			msgEvent,
			parts[1],
		)

	default:
		sendLotteryUsage(
			client,
			msgEvent,
		)
	}

	return true
}

func handleLotteryStatus(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	status, err :=
		lottery.GetStatus(
			groupJID,
			jid,
		)

	if err != nil {
		logger.Error(
			"Erro consultando !loteria:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível consultar a Loteria agora.",
		)

		return
	}

	response := fmt.Sprintf(
		"🎟️ *LOTERIA DO GRUPO*\n\n"+
			"🔢 Rodada: *#%d*\n"+
			"🏆 Jackpot atual: *%d Gold*\n\n"+
			"🎫 Bilhetes vendidos: *%d/%d*\n"+
			"🎫 Bilhetes restantes: *%d*\n\n"+
			"👤 Seus bilhetes: *%d*\n"+
			"🍀 Sua chance atual: *%.0f%%*\n\n"+
			"💰 Preço por bilhete: *%d Gold*\n"+
			"🎟️ Limite por jogador: *%d bilhetes*\n"+
			"🛒 Você ainda pode comprar: *%d*\n\n"+
			"🔥 *%d%%* das vendas vão para o Jackpot.\n"+
			"🔥 *%d%%* são removidos da economia.\n\n"+
			"Para comprar:\n"+
			"*!loteria <quantidade>*\n\n"+
			"Exemplo:\n"+
			"*!loteria 5*",
		status.Round.RoundNumber,
		status.Round.Jackpot,
		status.Round.TicketsSold,
		lottery.MaxTickets,
		status.RemainingTickets,
		status.UserTickets,
		status.ChancePercent,
		lottery.TicketPrice,
		lottery.MaxTicketsPerPlayer,
		status.UserCanBuy,
		lottery.JackpotPercent,
		100-lottery.JackpotPercent,
	)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}

func handleLotteryPurchase(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	quantityText string,
) {
	quantity, err :=
		strconv.Atoi(
			strings.TrimSpace(
				quantityText,
			),
		)

	if err != nil {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ A quantidade de bilhetes precisa ser um número inteiro.\n\n"+
				"Exemplo: *!loteria 5*",
		)

		return
	}

	if quantity <= 0 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ A quantidade precisa ser maior que zero.",
		)

		return
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	result, err :=
		lottery.BuyTickets(
			groupJID,
			jid,
			quantity,
		)

	if err != nil {
		handleLotteryPurchaseError(
			client,
			msgEvent,
			groupJID,
			jid,
			err,
		)

		return
	}

	if err :=
		profile.RecordLotteryPurchase(
			groupJID,
			jid,
			result.Quantity,
		); err != nil {

		logger.Error(
			"Erro registrando compra da Loteria no perfil:",
			err,
		)
	}

	if result.Drawn {
		if err :=
			profile.RecordLotteryWin(
				groupJID,
				result.WinnerJID,
				result.Prize,
			); err != nil {

			logger.Error(
				"Erro registrando vitória da Loteria no perfil:",
				err,
			)
		}
	}

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

	ticketText :=
		formatLotteryTicketRange(
			result.TicketFrom,
			result.TicketTo,
		)

	response := fmt.Sprintf(
		"@%s\n\n"+
			"🎟️ *BILHETES COMPRADOS!*\n\n"+
			"🎫 Bilhetes recebidos: *%s*\n"+
			"🎫 Quantidade: *%d*\n"+
			"💰 Custo: *%d Gold*\n\n"+
			"👤 Seus bilhetes nesta rodada: *%d*\n"+
			"🍀 Sua chance atual: *%.0f%%*\n\n"+
			"🏆 Jackpot: *%d Gold*\n"+
			"🎟️ Rodada: *%d/%d*\n"+
			"💰 Seu saldo: *%d Gold*",
		name,
		ticketText,
		result.Quantity,
		result.Cost,
		result.UserTickets,
		result.ChancePercent,
		result.Jackpot,
		result.TicketsSold,
		lottery.MaxTickets,
		result.Balance,
	)

	mentions :=
		[]types.JID{
			msgEvent.Info.
				Sender.
				ToNonAD(),
		}

	if result.Drawn {
		drawText, drawMentions :=
			renderLotteryDraw(
				result,
			)

		response +=
			"\n\n" +
				drawText

		for _, mention := range drawMentions {

			if !containsMention(
				mentions,
				mention,
			) {
				mentions =
					append(
						mentions,
						mention,
					)
			}
		}
	}

	err =
		whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			mentions,
		)

	if err != nil {
		logger.Error(
			"Erro enviando resultado do !loteria:",
			err,
		)

		return
	}

	logger.Info(
		"Loteria:",
		"Grupo:",
		groupJID,
		"Usuário:",
		jid,
		"Rodada:",
		result.RoundNumber,
		"Quantidade:",
		result.Quantity,
		"Custo:",
		result.Cost,
		"Bilhetes:",
		ticketText,
		"Jackpot:",
		result.Jackpot,
		"Sorteio:",
		result.Drawn,
	)
}

func handleLotteryPurchaseError(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	groupJID string,
	jid string,
	err error,
) {
	switch {

	case errors.Is(
		err,
		lottery.ErrInvalidQuantity,
	):
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"❌ Quantidade de bilhetes inválida.\n\n"+
					"Você pode comprar até *%d bilhetes por rodada*.\n\n"+
					"Exemplo: *!loteria 5*",
				lottery.MaxTicketsPerPlayer,
			),
		)

	case errors.Is(
		err,
		lottery.ErrWalletNotFound,
	):
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"❌ Você ainda não possui uma carteira Gold neste grupo.\n\n"+
					"Use *!gold* para receber seus *%d Gold* iniciais.",
				gold.InitialGold,
			),
		)

	case errors.Is(
		err,
		lottery.ErrInsufficientGold,
	):
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"❌ Você não possui Gold suficiente para comprar esses bilhetes.\n\n"+
					"💰 Cada bilhete custa *%d Gold*.",
				lottery.TicketPrice,
			),
		)

	case errors.Is(
		err,
		lottery.ErrPlayerTicketLimit,
	):
		status, statusErr :=
			lottery.GetStatus(
				groupJID,
				jid,
			)

		if statusErr == nil {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"❌ Você ultrapassaria o limite de bilhetes desta rodada.\n\n"+
						"🎫 Seus bilhetes: *%d/%d*\n"+
						"🛒 Você ainda pode comprar: *%d*",
					status.UserTickets,
					lottery.MaxTicketsPerPlayer,
					status.UserCanBuy,
				),
			)

			return
		}

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"❌ O limite é de *%d bilhetes por jogador em cada rodada*.",
				lottery.MaxTicketsPerPlayer,
			),
		)

	case errors.Is(
		err,
		lottery.ErrNotEnoughTickets,
	):
		status, statusErr :=
			lottery.GetStatus(
				groupJID,
				jid,
			)

		if statusErr == nil {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"❌ Não existem bilhetes suficientes disponíveis nesta rodada.\n\n"+
						"🎫 Restantes: *%d*\n"+
						"🛒 Você pode comprar agora: *%d*",
					status.RemainingTickets,
					status.UserCanBuy,
				),
			)

			return
		}

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não existem bilhetes suficientes disponíveis nesta rodada.",
		)

	default:
		logger.Error(
			"Erro comprando bilhetes da Loteria:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível comprar os bilhetes da Loteria agora.",
		)
	}
}

func renderLotteryDraw(
	result *lottery.PurchaseResult,
) (string, []types.JID) {
	winnerText :=
		result.WinnerJID

	mentions :=
		make(
			[]types.JID,
			0,
			1,
		)

	winnerJID, err :=
		types.ParseJID(
			result.WinnerJID,
		)

	if err == nil {
		winnerText =
			"@" +
				winnerJID.User

		mentions =
			append(
				mentions,
				winnerJID.ToNonAD(),
			)
	} else {
		logger.Error(
			"Erro convertendo JID do vencedor da Loteria:",
			err,
		)
	}

	text := fmt.Sprintf(
		"🎰 *TODOS OS BILHETES FORAM VENDIDOS!*\n\n"+
			"🎟️ *%d/%d*\n\n"+
			"🎲 Sorteio realizado!\n\n"+
			"🏆 *JACKPOT!*\n\n"+
			"🎫 Bilhete vencedor: *#%03d*\n"+
			"👑 Vencedor: *%s*\n"+
			"💰 Prêmio: *%d Gold*\n"+
			"💰 Saldo do vencedor: *%d Gold*\n\n"+
			"🆕 A rodada *#%d* já foi iniciada!\n"+
			"🎟️ Bilhetes: *0/%d*\n"+
			"🏆 Jackpot: *0 Gold*",
		lottery.MaxTickets,
		lottery.MaxTickets,
		result.WinningTicket,
		winnerText,
		result.Prize,
		result.WinnerBalance,
		result.NextRoundNumber,
		lottery.MaxTickets,
	)

	return text, mentions
}

func formatLotteryTicketRange(
	first int,
	last int,
) string {
	if first == last {
		return fmt.Sprintf(
			"#%03d",
			first,
		)
	}

	return fmt.Sprintf(
		"#%03d até #%03d",
		first,
		last,
	)
}

func containsMention(
	list []types.JID,
	target types.JID,
) bool {
	target =
		target.ToNonAD()

	for _, current := range list {

		if current.ToNonAD() ==
			target {

			return true
		}
	}

	return false
}

func sendLotteryUsage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	response := fmt.Sprintf(
		"🎟️ *LOTERIA DO GRUPO*\n\n"+
			"Para consultar a rodada:\n"+
			"*!loteria*\n\n"+
			"Para comprar bilhetes:\n"+
			"*!loteria <quantidade>*\n\n"+
			"Exemplo:\n"+
			"*!loteria 5*\n\n"+
			"💰 Cada bilhete custa *%d Gold*.\n"+
			"🎫 Limite: *%d bilhetes por jogador/rodada*.",
		lottery.TicketPrice,
		lottery.MaxTicketsPerPlayer,
	)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}
