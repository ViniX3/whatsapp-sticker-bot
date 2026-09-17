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
	// Ignora mensagens recebidas antes da inicialização atual do bot.
	if msgEvent.Info.Timestamp.Before(botStartedAt) {
		logger.Debug("Mensagem antiga ignorada")
		return
	}

	logger.Info("==============================")
	logger.Info("Mensagem recebida")

	// Validação de whitelist para grupos.
	if msgEvent.Info.IsGroup {
		groupID := msgEvent.Info.Chat.String()

		if !auth.IsGroupAllowed(groupID) {
			logger.Debug("Grupo não autorizado:", groupID)
			logger.Info("==============================")
			return
		}

		logger.Info("Grupo autorizado:", groupID)
	}

	text := extractText(msgEvent)
	logger.Info("Texto recebido:", text)

	parts := strings.Fields(text)

	if len(parts) == 0 {
		logger.Debug("Mensagem sem texto, ignorando")
		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !gold
	// ==========================================================

	if strings.EqualFold(parts[0], "!gold") {
		if len(parts) != 1 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!gold*",
			)

			logger.Info("==============================")
			return
		}

		jid := msgEvent.Info.Sender.String()
		name := msgEvent.Info.PushName

		logger.Info("Comando !gold recebido de:", jid)

		// Toda a criação da carteira, concessão do Gold inicial
		// e registro da transação ocorre atomicamente dentro
		// de gold.ClaimInitialGold().
		user, claimed, err := gold.ClaimInitialGold(jid, name)

		if err != nil {
			logger.Error(
				"Erro ao acessar/conceder Gold inicial:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Erro ao acessar carteira Gold.",
			)

			logger.Info("==============================")
			return
		}

		if user == nil {
			logger.Error(
				"Carteira Gold não encontrada após operação:",
				jid,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Erro ao consultar carteira Gold.",
			)

			logger.Info("==============================")
			return
		}

		// Gold concedido nesta chamada.
		if claimed {
			response := fmt.Sprintf(
				"🪙 *@%s*\n\n*Gold inicial recebido!*\n\nVocê recebeu *%d Gold* iniciais.\nSaldo atual: *%d Gold*",
				name,
				gold.InitialGold,
				user.Gold,
			)

			_ = whatsapp.SendMentionedText(
				client,
				msgEvent.Info.Chat,
				response,
				[]types.JID{msgEvent.Info.Sender},
			)

			logger.Success(
				"Gold inicial concedido para:",
				jid,
			)

			logger.Info("==============================")
			return
		}

		// Carteira já existia e o Gold inicial já havia sido recebido.
		response := fmt.Sprintf(
			"🪙 *@%s*\n\n*Carteira Gold*\n\nSaldo atual: *%d Gold*",
			name,
			user.Gold,
		)

		_ = whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{msgEvent.Info.Sender},
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

	if strings.EqualFold(parts[0], "!saldo") {
		if len(parts) != 1 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!saldo*",
			)

			logger.Info("==============================")
			return
		}

		jid := msgEvent.Info.Sender.String()
		name := msgEvent.Info.PushName

		user, err := database.GetUser(jid)

		if err != nil {
			logger.Error(
				"Erro ao consultar saldo:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Erro ao consultar saldo.",
			)

			logger.Info("==============================")
			return
		}

		if user == nil {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Você ainda não possui uma carteira Gold.\n\nUse *!gold* para receber seus *1000 Gold* iniciais.",
			)

			logger.Info("==============================")
			return
		}

		response := fmt.Sprintf(
			"💰 *@%s*\n\n*Seu saldo*\n\n*%d Gold*",
			name,
			user.Gold,
		)

		_ = whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{msgEvent.Info.Sender},
		)

		logger.Info(
			"Saldo consultado:",
			jid,
		)

		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !bet
	// ==========================================================

	if strings.EqualFold(parts[0], "!bet") {
		if len(parts) != 2 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!bet <quantidade>*\n\nExemplo: *!bet 100*",
			)

			logger.Info("==============================")
			return
		}

		betAmount, err := strconv.Atoi(parts[1])

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

		jid := msgEvent.Info.Sender.String()
		name := msgEvent.Info.PushName

		result, err := gold.Bet(
			jid,
			betAmount,
		)

		if err != nil {
			if errors.Is(
				err,
				gold.ErrWalletNotFound,
			) {
				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ Você ainda não possui uma carteira Gold.\n\nUse *!gold* primeiro.",
				)

				logger.Info("==============================")
				return
			}

			if errors.Is(
				err,
				gold.ErrInsufficientGold,
			) {
				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ Você não possui Gold suficiente para essa aposta.",
				)

				logger.Info("==============================")
				return
			}

			logger.Error(
				"Erro ao realizar aposta:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Erro ao realizar aposta.",
			)

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
			[]types.JID{msgEvent.Info.Sender},
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

	if strings.EqualFold(parts[0], "!roubar") {
		jid := msgEvent.Info.Sender.String()
		robberName := msgEvent.Info.PushName

		if !msgEvent.Info.IsGroup {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ O comando *!roubar* só pode ser usado em grupos.",
			)

			logger.Info("==============================")
			return
		}

		if len(parts) != 2 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!roubar @pessoa*",
			)

			logger.Info("==============================")
			return
		}

		if msgEvent.Message == nil ||
			msgEvent.Message.ExtendedTextMessage == nil {

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Você precisa mencionar alguém para roubar.",
			)

			logger.Info("==============================")
			return
		}

		contextInfo :=
			msgEvent.Message.
				ExtendedTextMessage.
				ContextInfo

		if contextInfo == nil {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Você precisa mencionar alguém para roubar.",
			)

			logger.Info("==============================")
			return
		}

		mentionedJIDs :=
			contextInfo.GetMentionedJID()

		if len(mentionedJIDs) == 0 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Você precisa mencionar alguém para roubar.",
			)

			logger.Info("==============================")
			return
		}

		if len(mentionedJIDs) > 1 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Mencione apenas uma pessoa por vez.",
			)

			logger.Info("==============================")
			return
		}

		targetJID := mentionedJIDs[0]
		targetName :=
			strings.TrimPrefix(
				parts[1],
				"@",
			)

		if targetJID == jid {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"😂 Você não pode roubar a si mesmo.",
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

		targetInGroup := false

		for _, participant := range groupInfo.Participants {

			if participant.JID.String() ==
				targetJID {

				targetInGroup = true
				break
			}
		}

		if !targetInGroup {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ A pessoa mencionada não pertence a este grupo.",
			)

			logger.Info("==============================")
			return
		}

		// Consulta antecipada apenas para fornecer uma mensagem
		// amigável ao usuário.
		//
		// A validação definitiva e atômica do cooldown acontece
		// novamente dentro de gold.Rob().
		if !gold.CanRob(jid) {
			remaining :=
				gold.GetRobCooldown(jid)

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
				jid,
				targetJID,
			)

		if err != nil {
			if errors.Is(
				err,
				gold.ErrWalletNotFound,
			) {
				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ Você precisa criar sua carteira primeiro usando *!gold*.",
				)

				logger.Info("==============================")
				return
			}

			if errors.Is(
				err,
				gold.ErrInsufficientGold,
			) {
				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ Você não possui Gold suficiente.",
				)

				logger.Info("==============================")
				return
			}

			if errors.Is(
				err,
				gold.ErrRobCooldown,
			) {
				remaining :=
					gold.GetRobCooldown(jid)

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

				logger.Info("==============================")
				return
			}

			logger.Error(
				"Erro ao executar roubo:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Ocorreu um erro ao tentar realizar o roubo.",
			)

			logger.Info("==============================")
			return
		}

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

		_ = whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{
				msgEvent.Info.Sender,
				targetJIDParsed,
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

// extractText obtém o texto ou legenda da mensagem.
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

// getMedia identifica mídia enviada diretamente ou através
// de uma mensagem respondida.
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

// processStickerCommand executa o pipeline de geração e envio
// da figurinha.
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
