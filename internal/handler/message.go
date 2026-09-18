package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"whatsapp-sticker-bot/internal/auth"
	"whatsapp-sticker-bot/internal/database"
	"whatsapp-sticker-bot/internal/gold"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/media"
	"whatsapp-sticker-bot/internal/processor"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func ProcessMessage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	botStartedAt time.Time,
) {
	if msgEvent.Info.Timestamp.Before(botStartedAt) {
		logger.Debug("Mensagem antiga ignorada")
		return
	}

	logger.Info("==============================")
	logger.Info("Mensagem recebida")

	// ==========================================================
	// WHITELIST DE GRUPOS
	// ==========================================================

	if msgEvent.Info.IsGroup {
		groupID := msgEvent.Info.Chat.String()

		if !auth.IsGroupAllowed(groupID) {
			logger.Debug(
				"Grupo não autorizado:",
				groupID,
			)

			logger.Info("==============================")
			return
		}

		logger.Info(
			"Grupo autorizado:",
			groupID,
		)
	}

	text := extractText(msgEvent)

	logger.Info(
		"Texto recebido:",
		text,
	)

	parts := strings.Fields(text)

	if len(parts) == 0 {
		logger.Debug(
			"Mensagem sem texto, ignorando",
		)

		logger.Info("==============================")
		return
	}

	command := strings.ToLower(parts[0])

	// ==========================================================
	// !gold
	// ==========================================================

	if command == "!gold" {
		if !requireGoldGroup(
			client,
			msgEvent,
			"!gold",
		) {
			return
		}

		if len(parts) != 1 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!gold*",
			)

			logger.Info("==============================")
			return
		}

		groupJID :=
			msgEvent.Info.Chat.String()

		jid :=
			canonicalSenderJID(msgEvent)

		name :=
			msgEvent.Info.PushName

		logger.Info(
			"Comando !gold recebido de:",
			jid,
		)

		wallet, claimed, err :=
			gold.ClaimInitialGold(
				groupJID,
				jid,
				name,
			)

		if err != nil {
			logger.Error(
				"Erro ao acessar/conceder Gold inicial:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Erro ao acessar sua carteira Gold.",
			)

			logger.Info("==============================")
			return
		}

		if wallet == nil {
			logger.Error(
				"Carteira Gold não encontrada após operação:",
				jid,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Erro ao consultar sua carteira Gold.",
			)

			logger.Info("==============================")
			return
		}

		senderMention :=
			msgEvent.Info.Sender.ToNonAD()

		if claimed {
			response := fmt.Sprintf(
				"🪙 *@%s*\n\n*Carteira Gold ativada neste grupo!*\n\nVocê recebeu *%d Gold* iniciais.\n\n💰 Saldo atual: *%d Gold*",
				name,
				gold.InitialGold,
				wallet.Gold,
			)

			_ = whatsapp.SendMentionedText(
				client,
				msgEvent.Info.Chat,
				response,
				[]types.JID{
					senderMention,
				},
			)

			logger.Success(
				"Gold inicial concedido no grupo:",
				groupJID,
				"Usuário:",
				jid,
			)

			logger.Info("==============================")
			return
		}

		response := fmt.Sprintf(
			"🪙 *@%s*\n\n*Carteira Gold deste grupo*\n\n💰 Saldo atual: *%d Gold*",
			name,
			wallet.Gold,
		)

		_ = whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{
				senderMention,
			},
		)

		logger.Info(
			"Carteira Gold consultada:",
			jid,
		)

		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !saldo
	// ==========================================================

	if command == "!saldo" {
		if !requireGoldGroup(
			client,
			msgEvent,
			"!saldo",
		) {
			return
		}

		if len(parts) != 1 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!saldo*",
			)

			logger.Info("==============================")
			return
		}

		groupJID :=
			msgEvent.Info.Chat.String()

		jid :=
			canonicalSenderJID(msgEvent)

		name :=
			msgEvent.Info.PushName

		wallet, err :=
			database.GetWallet(
				groupJID,
				jid,
			)

		if err != nil {
			logger.Error(
				"Erro ao consultar saldo:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Erro ao consultar seu saldo.",
			)

			logger.Info("==============================")
			return
		}

		if wallet == nil {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"❌ Você ainda não possui uma carteira Gold neste grupo.\n\nUse *!gold* para receber seus *%d Gold* iniciais.",
					gold.InitialGold,
				),
			)

			logger.Info("==============================")
			return
		}

		response := fmt.Sprintf(
			"💰 *@%s*\n\n*Seu saldo neste grupo*\n\n*%d Gold*",
			name,
			wallet.Gold,
		)

		_ = whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{
				msgEvent.Info.Sender.ToNonAD(),
			},
		)

		logger.Info(
			"Saldo consultado:",
			jid,
		)

		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !escudo
	// ==========================================================

	if command == "!escudo" {
		if !requireGoldGroup(
			client,
			msgEvent,
			"!escudo",
		) {
			return
		}

		if len(parts) != 1 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!escudo*",
			)

			logger.Info("==============================")
			return
		}

		groupJID :=
			msgEvent.Info.Chat.String()

		jid :=
			canonicalSenderJID(msgEvent)

		name :=
			msgEvent.Info.PushName

		result, err :=
			gold.BuyShield(
				groupJID,
				jid,
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
					fmt.Sprintf(
						"💸 Você precisa de *%d Gold* para comprar um escudo.",
						gold.ShieldPrice,
					),
				)

			case errors.Is(
				err,
				gold.ErrShieldAlreadyActive,
			):
				status, statusErr :=
					gold.GetShieldStatus(
						groupJID,
						jid,
					)

				if statusErr != nil {
					logger.Error(
						"Erro ao consultar escudo ativo:",
						statusErr,
					)

					_ = whatsapp.SendText(
						client,
						msgEvent.Info.Chat,
						"🛡️ Você já possui um escudo ativo neste grupo.",
					)

					logger.Info("==============================")
					return
				}

				if status != nil &&
					status.Active {

					response := fmt.Sprintf(
						"🛡️ @%s, você já possui um escudo ativo!\n\n⏳ Tempo restante: *%s*\n⚔️ Ataques recebidos: *%d*\n\nUm novo escudo só poderá ser comprado quando este quebrar ou expirar.",
						name,
						formatDuration(
							status.Remaining,
						),
						status.AttacksReceived,
					)

					_ = whatsapp.SendMentionedText(
						client,
						msgEvent.Info.Chat,
						response,
						[]types.JID{
							msgEvent.Info.Sender.ToNonAD(),
						},
					)

					logger.Info("==============================")
					return
				}

				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"🛡️ Você já possui um escudo ativo neste grupo.",
				)

			default:
				logger.Error(
					"Erro ao comprar escudo:",
					err,
				)

				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ Não foi possível comprar o escudo.",
				)
			}

			logger.Info("==============================")
			return
		}

		response := fmt.Sprintf(
			"🛡️ *ESCUDO ATIVADO!*\n\n*@%s* agora está protegido contra roubos.\n\n💰 Custo: *%d Gold*\n⏳ Duração máxima: *12 horas*\n💰 Saldo atual: *%d Gold*\n\n⚔️ Cada tentativa de roubo desgasta o escudo e existe uma pequena chance de ele quebrar.",
			name,
			result.Price,
			result.Balance,
		)

		_ = whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{
				msgEvent.Info.Sender.ToNonAD(),
			},
		)

		logger.Success(
			"Escudo comprado:",
			jid,
			"Grupo:",
			groupJID,
			"Expira:",
			result.ExpiresAt,
		)

		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !sorte
	// ==========================================================

	if command == "!sorte" {
		if !requireGoldGroup(
			client,
			msgEvent,
			"!sorte",
		) {
			return
		}

		if len(parts) != 1 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!sorte*",
			)

			logger.Info("==============================")
			return
		}

		groupJID :=
			msgEvent.Info.Chat.String()

		jid :=
			canonicalSenderJID(msgEvent)

		name :=
			msgEvent.Info.PushName

		result, err :=
			gold.ClaimDailyLuck(
				groupJID,
				jid,
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
				gold.ErrLuckCooldown,
			):
				var cooldownErr *gold.LuckCooldownError

				if errors.As(
					err,
					&cooldownErr,
				) {
					response := fmt.Sprintf(
						"🍀 @%s, você já tentou sua sorte hoje!*\n\n⏳ Próxima tentativa em: *%s*",
						name,
						formatDuration(
							cooldownErr.Remaining,
						),
					)

					_ = whatsapp.SendMentionedText(
						client,
						msgEvent.Info.Chat,
						response,
						[]types.JID{
							msgEvent.Info.Sender.ToNonAD(),
						},
					)
				} else {
					_ = whatsapp.SendText(
						client,
						msgEvent.Info.Chat,
						"🍀 Você já tentou sua sorte hoje. Tente novamente mais tarde.",
					)
				}

			default:
				logger.Error(
					"Erro ao executar sorte diária:",
					err,
				)

				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ Não foi possível tentar sua sorte agora.",
				)
			}

			logger.Info("==============================")
			return
		}

		headline :=
			luckHeadline(
				result.Tier,
			)

		response := fmt.Sprintf(
			"%s\n\n🍀 *@%s* tirou:\n\n%s *%s*\n\n💰 Prêmio: *+%d Gold*\n💰 Saldo atual: *%d Gold*\n\n⏳ Você poderá tentar novamente em *24 horas*.",
			headline,
			name,
			result.Emoji,
			result.Tier,
			result.Amount,
			result.Balance,
		)

		_ = whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{
				msgEvent.Info.Sender.ToNonAD(),
			},
		)

		logger.Success(
			"Sorte diária realizada:",
			jid,
			"Grupo:",
			groupJID,
			"Raridade:",
			result.Tier,
			"Prêmio:",
			result.Amount,
		)

		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !ranking
	// ==========================================================

	if command == "!ranking" {
		if !requireGoldGroup(
			client,
			msgEvent,
			"!ranking",
		) {
			return
		}

		if len(parts) != 1 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!ranking*",
			)

			logger.Info("==============================")
			return
		}

		groupJID :=
			msgEvent.Info.Chat.String()

		ranking, err :=
			gold.GetRanking(
				groupJID,
				10,
			)

		if err != nil {
			logger.Error(
				"Erro ao consultar ranking:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível consultar o ranking Gold.",
			)

			logger.Info("==============================")
			return
		}

		if len(ranking) == 0 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"🏆 Ainda não existem carteiras Gold neste grupo.",
			)

			logger.Info("==============================")
			return
		}

		var builder strings.Builder

		builder.WriteString(
			"🏆 *RANKING GOLD DO GRUPO*\n\n",
		)

		for position, entry := range ranking {

			var prefix string

			switch position {
			case 0:
				prefix = "🥇"
			case 1:
				prefix = "🥈"
			case 2:
				prefix = "🥉"
			default:
				prefix = fmt.Sprintf(
					"%d.",
					position+1,
				)
			}

			name :=
				rankingDisplayName(
					entry.Name,
					entry.JID,
				)

			builder.WriteString(
				fmt.Sprintf(
					"%s *%s* — %d Gold\n",
					prefix,
					name,
					entry.Gold,
				),
			)
		}

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			builder.String(),
		)

		logger.Info(
			"Ranking consultado no grupo:",
			groupJID,
		)

		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !pix
	// ==========================================================

	if command == "!pix" {
		if !requireGoldGroup(
			client,
			msgEvent,
			"!pix",
		) {
			return
		}

		if len(parts) < 3 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!pix @pessoa <quantidade>*\n\nExemplo: *!pix @pessoa 500*",
			)

			logger.Info("==============================")
			return
		}

		targetJID, ok :=
			getSingleMention(msgEvent)

		if !ok {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Você precisa mencionar exatamente uma pessoa para enviar o PIX.",
			)

			logger.Info("==============================")
			return
		}

		amountText :=
			parts[len(parts)-1]

		amount, err :=
			strconv.Atoi(amountText)

		if err != nil ||
			amount <= 0 {

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ A quantidade do PIX precisa ser um número inteiro maior que zero.\n\nExemplo: *!pix @pessoa 500*",
			)

			logger.Info("==============================")
			return
		}

		groupJID :=
			msgEvent.Info.Chat.String()

		senderJID :=
			canonicalSenderJID(msgEvent)

		if targetJID == senderJID {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"😂 Você não pode fazer um PIX para si mesmo.",
			)

			logger.Info("==============================")
			return
		}

		groupInfo, err :=
			client.GetGroupInfo(
				context.Background(),
				msgEvent.Info.Chat,
			)

		if err != nil {
			logger.Error(
				"Erro ao buscar informações do grupo para PIX:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível verificar o destinatário do PIX.",
			)

			logger.Info("==============================")
			return
		}

		targetJID, ok =
			resolveParticipantJID(
				groupInfo.Participants,
				targetJID,
			)

		if !ok {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ A pessoa mencionada não pertence a este grupo.",
			)

			logger.Info("==============================")
			return
		}

		targetName :=
			pixTargetName(parts)

		result, err :=
			gold.Pix(
				groupJID,
				senderJID,
				targetJID,
				targetName,
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
					"💸 Você não possui Gold suficiente para realizar esse PIX.",
				)

			case errors.Is(
				err,
				gold.ErrPixSelfTransfer,
			):
				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"😂 Você não pode fazer um PIX para si mesmo.",
				)

			default:
				logger.Error(
					"Erro ao realizar PIX:",
					err,
				)

				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ Não foi possível realizar o PIX.",
				)
			}

			logger.Info("==============================")
			return
		}

		targetJIDParsed, err :=
			types.ParseJID(targetJID)

		if err != nil {
			logger.Error(
				"Erro ao converter JID do destinatário do PIX:",
				err,
			)

			logger.Info("==============================")
			return
		}

		senderName :=
			msgEvent.Info.PushName

		response := fmt.Sprintf(
			"💸 *PIX GOLD REALIZADO!*\n\n*@%s* enviou *%d Gold* para *@%s*.\n\n💰 Saldo de @%s: *%d Gold*",
			senderName,
			result.Amount,
			targetName,
			senderName,
			result.SenderBalance,
		)

		_ = whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{
				msgEvent.Info.Sender.ToNonAD(),
				targetJIDParsed.ToNonAD(),
			},
		)

		logger.Success(
			"PIX Gold realizado:",
			senderJID,
			"->",
			targetJID,
			"Valor:",
			result.Amount,
		)

		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !bet
	// ==========================================================

	if command == "!bet" {
		if !requireGoldGroup(
			client,
			msgEvent,
			"!bet",
		) {
			return
		}

		if len(parts) != 2 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!bet <quantidade>*\n\nExemplo: *!bet 100*",
			)

			logger.Info("==============================")
			return
		}

		betAmount, err :=
			strconv.Atoi(parts[1])

		if err != nil {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ A quantidade precisa ser um número inteiro.\n\nExemplo: *!bet 100*",
			)

			logger.Info("==============================")
			return
		}

		if betAmount <= 0 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ A aposta precisa ser maior que *0 Gold*.",
			)

			logger.Info("==============================")
			return
		}

		groupJID :=
			msgEvent.Info.Chat.String()

		jid :=
			canonicalSenderJID(msgEvent)

		name :=
			msgEvent.Info.PushName

		result, err :=
			gold.Bet(
				groupJID,
				jid,
				betAmount,
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

			default:
				logger.Error(
					"Erro ao realizar aposta:",
					err,
				)

				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ Erro ao realizar aposta.",
				)
			}

			logger.Info("==============================")
			return
		}

		var response string

		switch result.Multiplier {
		case 0:
			response = fmt.Sprintf(
				"💀 *PERDEU*\n\nAposta: *%d Gold*\nPrêmio: *0 Gold*\n\n💰 Saldo: *%d Gold*",
				result.BetAmount,
				result.Balance,
			)

		case 1:
			response = fmt.Sprintf(
				"😐 *RECUPEROU*\n\nAposta: *%d Gold*\nPrêmio: *%d Gold*\n\n💰 Saldo: *%d Gold*",
				result.BetAmount,
				result.Prize,
				result.Balance,
			)

		case 2:
			response = fmt.Sprintf(
				"🍀 *PEQUENO PRÊMIO*\n\nAposta: *%d Gold*\nPrêmio: *%d Gold*\nLucro: *+%d Gold*\n\n💰 Saldo: *%d Gold*",
				result.BetAmount,
				result.Prize,
				result.NetResult,
				result.Balance,
			)

		case 5:
			response = fmt.Sprintf(
				"💰 *GRANDE PRÊMIO!*\n\nAposta: *%d Gold*\nPrêmio: *%d Gold*\nLucro: *+%d Gold*\n\n💰 Saldo: *%d Gold*",
				result.BetAmount,
				result.Prize,
				result.NetResult,
				result.Balance,
			)

		case 10:
			response = fmt.Sprintf(
				"🔥 *JACKPOT!*\n\nAposta: *%d Gold*\nPrêmio: *%d Gold*\nLucro: *+%d Gold*\n\n💰 Saldo: *%d Gold*",
				result.BetAmount,
				result.Prize,
				result.NetResult,
				result.Balance,
			)

		case 50:
			response = fmt.Sprintf(
				"👑 *MEGA JACKPOT!!!*\n\nAposta: *%d Gold*\nPrêmio: *%d Gold*\nLucro: *+%d Gold*\n\n💰 Saldo: *%d Gold*",
				result.BetAmount,
				result.Prize,
				result.NetResult,
				result.Balance,
			)

		default:
			response = fmt.Sprintf(
				"🎲 Resultado: *%dx*\n\nAposta: *%d Gold*\nPrêmio: *%d Gold*\n\n💰 Saldo: *%d Gold*",
				result.Multiplier,
				result.BetAmount,
				result.Prize,
				result.Balance,
			)
		}

		response = fmt.Sprintf(
			"@%s\n\n%s",
			name,
			response,
		)

		_ = whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{
				msgEvent.Info.Sender.ToNonAD(),
			},
		)

		logger.Info(
			"Aposta realizada por:",
			jid,
		)

		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !roubar
	// ==========================================================

	if command == "!roubar" {
		if !requireGoldGroup(
			client,
			msgEvent,
			"!roubar",
		) {
			return
		}

		groupJID :=
			msgEvent.Info.Chat.String()

		jid :=
			canonicalSenderJID(msgEvent)

		robberName :=
			msgEvent.Info.PushName

		if len(parts) < 2 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!roubar @pessoa*",
			)

			logger.Info("==============================")
			return
		}

		targetJID, ok :=
			getSingleMention(msgEvent)

		if !ok {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Você precisa mencionar exatamente uma pessoa para roubar.",
			)

			logger.Info("==============================")
			return
		}

		targetName :=
			robTargetName(parts)

		groupInfo, err :=
			client.GetGroupInfo(
				context.Background(),
				msgEvent.Info.Chat,
			)

		if err != nil {
			logger.Error(
				"Erro ao buscar informações do grupo:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível verificar o participante.",
			)

			logger.Info("==============================")
			return
		}

		targetJID, ok =
			resolveParticipantJID(
				groupInfo.Participants,
				targetJID,
			)

		if !ok {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ A pessoa mencionada não pertence a este grupo.",
			)

			logger.Info("==============================")
			return
		}

		if targetJID == jid {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"😂 Você não pode roubar a si mesmo.",
			)

			logger.Info("==============================")
			return
		}

		targetJIDParsed, err :=
			types.ParseJID(targetJID)

		if err != nil {
			logger.Error(
				"Erro ao converter JID do alvo:",
				err,
			)

			logger.Info("==============================")
			return
		}

		if !gold.CanRob(
			groupJID,
			jid,
		) {
			remaining :=
				gold.GetRobCooldown(
					groupJID,
					jid,
				)

			seconds :=
				int(remaining.Seconds())

			if seconds < 1 {
				seconds = 1
			}

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"⏳ Calma! Você precisa esperar *%d segundos* para tentar roubar novamente.",
					seconds,
				),
			)

			logger.Info("==============================")
			return
		}

		result, err :=
			gold.Rob(
				groupJID,
				jid,
				targetJID,
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
				gold.ErrTargetNoGold,
			):
				response := fmt.Sprintf(
					"💸 *@%s* não possui Gold disponível para ser roubado.",
					targetName,
				)

				_ = whatsapp.SendMentionedText(
					client,
					msgEvent.Info.Chat,
					response,
					[]types.JID{
						targetJIDParsed.ToNonAD(),
					},
				)

			case errors.Is(
				err,
				gold.ErrRobCooldown,
			):
				remaining :=
					gold.GetRobCooldown(
						groupJID,
						jid,
					)

				seconds :=
					int(remaining.Seconds())

				if seconds < 1 {
					seconds = 1
				}

				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					fmt.Sprintf(
						"⏳ Espere *%d segundos* antes de tentar roubar novamente.",
						seconds,
					),
				)

			default:
				logger.Error(
					"Erro ao executar roubo:",
					err,
				)

				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ Não foi possível concluir o roubo. Tente novamente.",
				)
			}

			logger.Info("==============================")
			return
		}

		// ======================================================
		// ESCUDO BLOQUEOU O ROUBO
		// ======================================================

		if result.ShieldBlocked {
			if result.ShieldBroken {
				response := fmt.Sprintf(
					"💥 *ESCUDO QUEBRADO!*\n\nO ataque de *@%s* destruiu o escudo de *@%s*!\n\n⚔️ O escudo quebrou no ataque nº *%d*.\n\n💰 Nenhum Gold foi roubado nesta tentativa.\n\n🛡️ @%s pode comprar outro escudo imediatamente usando *!escudo*.",
					robberName,
					targetName,
					result.ShieldAttackNumber,
					targetName,
				)

				_ = whatsapp.SendMentionedText(
					client,
					msgEvent.Info.Chat,
					response,
					[]types.JID{
						msgEvent.Info.Sender.ToNonAD(),
						targetJIDParsed.ToNonAD(),
					},
				)

				logger.Warn(
					"Escudo quebrado:",
					"Alvo:",
					targetJID,
					"Ataque:",
					result.ShieldAttackNumber,
					"Chance:",
					result.ShieldBreakChance,
					"%",
				)

				logger.Info("==============================")
				return
			}

			response := fmt.Sprintf(
				"🛡️ *ATAQUE BLOQUEADO!*\n\nO escudo de *@%s* resistiu ao ataque de *@%s*.\n\n⚔️ Ataques recebidos por este escudo: *%d*\n💰 Nenhum Gold foi roubado.",
				targetName,
				robberName,
				result.ShieldAttackNumber,
			)

			_ = whatsapp.SendMentionedText(
				client,
				msgEvent.Info.Chat,
				response,
				[]types.JID{
					msgEvent.Info.Sender.ToNonAD(),
					targetJIDParsed.ToNonAD(),
				},
			)

			logger.Info(
				"Escudo bloqueou roubo:",
				"Alvo:",
				targetJID,
				"Ataque:",
				result.ShieldAttackNumber,
				"Chance quebra:",
				result.ShieldBreakChance,
				"%",
			)

			logger.Info("==============================")
			return
		}

		// ======================================================
		// ROUBO NORMAL
		// ======================================================

		var response string

		if result.Success {
			response = fmt.Sprintf(
				"🦹 *@%s* roubou *%d Gold* de *@%s*! 💰\n\n💰 Saldo de @%s: *%d Gold*",
				robberName,
				result.Amount,
				targetName,
				robberName,
				result.RobberBalance,
			)
		} else {
			response = fmt.Sprintf(
				"🚨 *@%s* tentou roubar *@%s*, mas falhou! 💸\n\nPerdeu *%d Gold*.\n\n💰 Saldo de @%s: *%d Gold*",
				robberName,
				targetName,
				result.Penalty,
				robberName,
				result.RobberBalance,
			)
		}

		_ = whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{
				msgEvent.Info.Sender.ToNonAD(),
				targetJIDParsed.ToNonAD(),
			},
		)

		logger.Info(
			"Tentativa de roubo realizada por:",
			jid,
		)

		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !f
	// ==========================================================

	if !IsStickerCommand(text) {
		logger.Debug(
			"Sem comando reconhecido, ignorando",
		)

		logger.Info("==============================")
		return
	}

	logger.Info("Comando !f recebido")

	mediaMessage :=
		getMedia(msgEvent)

	if mediaMessage == nil {
		logger.Warn(
			"Nenhuma mídia encontrada",
		)

		logger.Info("==============================")
		return
	}

	processStickerCommand(
		client,
		msgEvent,
		mediaMessage,
	)

	logger.Info("==============================")
}

// ==========================================================
// NORMALIZAÇÃO DE IDENTIDADE
// ==========================================================

func canonicalSenderJID(
	msg *events.Message,
) string {

	return msg.Info.
		Sender.
		ToNonAD().
		String()
}

func normalizeJIDString(
	value string,
) (string, bool) {

	jid, err :=
		types.ParseJID(value)

	if err != nil {
		return "", false
	}

	return jid.
		ToNonAD().
		String(), true
}

func resolveParticipantJID(
	participants []types.GroupParticipant,
	targetJID string,
) (string, bool) {

	target, ok :=
		normalizeJIDString(targetJID)

	if !ok {
		return "", false
	}

	for _, participant := range participants {

		candidates := []types.JID{
			participant.JID,
			participant.LID,
			participant.PhoneNumber,
		}

		for _, candidate := range candidates {

			if candidate.IsEmpty() {
				continue
			}

			normalized :=
				candidate.
					ToNonAD().
					String()

			if normalized == target {
				if !participant.JID.IsEmpty() {
					return participant.
						JID.
						ToNonAD().
						String(), true
				}

				return normalized, true
			}
		}
	}

	return "", false
}

// ==========================================================
// HELPERS DOS COMANDOS GOLD
// ==========================================================

func requireGoldGroup(
	client *whatsmeow.Client,
	msg *events.Message,
	command string,
) bool {

	if msg.Info.IsGroup {
		return true
	}

	_ = whatsapp.SendText(
		client,
		msg.Info.Chat,
		fmt.Sprintf(
			"❌ O comando *%s* só pode ser usado em grupos autorizados.",
			command,
		),
	)

	logger.Info("==============================")

	return false
}

func getSingleMention(
	msg *events.Message,
) (string, bool) {

	if msg.Message == nil ||
		msg.Message.ExtendedTextMessage == nil {

		return "", false
	}

	contextInfo :=
		msg.Message.
			ExtendedTextMessage.
			ContextInfo

	if contextInfo == nil {
		return "", false
	}

	mentioned :=
		contextInfo.GetMentionedJID()

	if len(mentioned) != 1 {
		return "", false
	}

	return normalizeJIDString(
		mentioned[0],
	)
}

func pixTargetName(
	parts []string,
) string {

	if len(parts) < 3 {
		return "usuário"
	}

	name :=
		strings.Join(
			parts[1:len(parts)-1],
			" ",
		)

	name =
		strings.TrimSpace(
			strings.TrimPrefix(
				name,
				"@",
			),
		)

	if name == "" {
		return "usuário"
	}

	return name
}

func robTargetName(
	parts []string,
) string {

	if len(parts) < 2 {
		return "usuário"
	}

	name :=
		strings.Join(
			parts[1:],
			" ",
		)

	name =
		strings.TrimSpace(
			strings.TrimPrefix(
				name,
				"@",
			),
		)

	if name == "" {
		return "usuário"
	}

	return name
}

func rankingDisplayName(
	name string,
	jid string,
) string {

	name =
		strings.TrimSpace(name)

	if name != "" {
		return name
	}

	if index :=
		strings.Index(
			jid,
			"@",
		); index > 0 {

		return jid[:index]
	}

	return jid
}

// ==========================================================
// FORMATAÇÃO DE TEMPO
// ==========================================================

// formatDuration é utilizada tanto pelo !escudo quanto
// pelo cooldown do !sorte.
//
// Exemplos:
//
//	23h 41m
//	11h 05m
//	53m
//	menos de 1 minuto
func formatDuration(
	duration time.Duration,
) string {

	if duration <= 0 {
		return "disponível agora"
	}

	totalMinutes :=
		int(duration.Minutes())

	if totalMinutes < 1 {
		return "menos de 1 minuto"
	}

	hours :=
		totalMinutes / 60

	minutes :=
		totalMinutes % 60

	if hours > 0 &&
		minutes > 0 {

		return fmt.Sprintf(
			"%dh %02dm",
			hours,
			minutes,
		)
	}

	if hours > 0 {
		return fmt.Sprintf(
			"%dh",
			hours,
		)
	}

	return fmt.Sprintf(
		"%dm",
		minutes,
	)
}

// ==========================================================
// FORMATAÇÃO DO !SORTE
// ==========================================================

func luckHeadline(
	tier string,
) string {

	switch tier {
	case "COMUM":
		return "🍀 *SORTE DO DIA!*"

	case "ESPECIAL":
		return "✨ *UMA BOA SORTE!*"

	case "RARO":
		return "💎 *QUE SORTE!*"

	case "SUPER RARO":
		return "🔥 *SUPER SORTE!*"

	case "LENDÁRIO":
		return "🌟 *SORTE LENDÁRIA!!!*"

	case "MÍTICO":
		return "👑 *SORTE MÍTICA!!!* 👑"

	default:
		return "🍀 *SORTE DO DIA!*"
	}
}

// ==========================================================
// TEXTO / MÍDIA
// ==========================================================

func extractText(
	msg *events.Message,
) string {

	if msg.Message.GetConversation() != "" {
		return strings.TrimSpace(
			msg.Message.GetConversation(),
		)
	}

	if msg.Message.ExtendedTextMessage != nil {
		return strings.TrimSpace(
			msg.Message.
				GetExtendedTextMessage().
				GetText(),
		)
	}

	if msg.Message.ImageMessage != nil {
		return strings.TrimSpace(
			msg.Message.
				GetImageMessage().
				GetCaption(),
		)
	}

	if msg.Message.VideoMessage != nil {
		return strings.TrimSpace(
			msg.Message.
				GetVideoMessage().
				GetCaption(),
		)
	}

	return ""
}

func getMedia(
	msg *events.Message,
) *media.Media {

	if msg.Message.ImageMessage != nil {
		return &media.Media{
			Type:  media.Image,
			Image: msg.Message.ImageMessage,
		}
	}

	if msg.Message.VideoMessage != nil {
		return &media.Media{
			Type:  media.Video,
			Video: msg.Message.VideoMessage,
		}
	}

	if msg.Message.ExtendedTextMessage != nil {
		contextInfo :=
			msg.Message.
				ExtendedTextMessage.
				ContextInfo

		if contextInfo != nil &&
			contextInfo.QuotedMessage != nil {

			if contextInfo.
				QuotedMessage.
				ImageMessage != nil {

				return &media.Media{
					Type: media.Image,
					Image: contextInfo.
						QuotedMessage.
						ImageMessage,
				}
			}

			if contextInfo.
				QuotedMessage.
				VideoMessage != nil {

				return &media.Media{
					Type: media.Video,
					Video: contextInfo.
						QuotedMessage.
						VideoMessage,
				}
			}
		}
	}

	return nil
}

func processStickerCommand(
	client *whatsmeow.Client,
	msg *events.Message,
	mediaMessage *media.Media,
) {

	err := processor.ProcessSticker(
		client,
		msg.Info.Chat,
		mediaMessage,
	)

	if err != nil {
		logger.Error(
			"Erro ao processar figurinha:",
			err,
		)

		return
	}

	logger.Success(
		"Processamento concluído",
	)
}
