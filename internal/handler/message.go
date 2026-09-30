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
	"whatsapp-sticker-bot/internal/quiz"
	"whatsapp-sticker-bot/internal/rpg"
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

	rawText := extractText(msgEvent)

	text :=
		normalizeCommandText(
			rawText,
		)

	logger.Info(
		"Texto recebido:",
		rawText,
	)

	if rawText != text {
		logger.Info(
			"Comando normalizado:",
			rawText,
			"->",
			text,
		)
	}

	parts := strings.Fields(text)

	if len(parts) == 0 {
		logger.Debug(
			"Mensagem sem texto, ignorando",
		)

		logger.Info("==============================")
		return
	}

	command := canonicalCommand(parts[0])

	// ==========================================================
	// SINCRONIZAÇÃO DE IDENTIDADE
	// ==========================================================
	//
	// Se o usuário já possuir identidade no banco,
	// atualizamos seu nome usando o PushName atual.
	//
	// Isso NÃO cria carteira e NÃO concede Gold.
	// ==========================================================

	if msgEvent.Info.IsGroup {
		senderJID :=
			canonicalSenderJID(
				msgEvent,
			)

		senderName :=
			strings.TrimSpace(
				msgEvent.Info.PushName,
			)

		if err :=
			database.UpdateUserNameIfExists(
				senderJID,
				senderName,
			); err != nil {

			logger.Warn(
				"Não foi possível sincronizar nome do usuário:",
				senderJID,
				err,
			)
		}
	}

	// ==========================================================
	// RESPOSTA DE QUIZ
	// ==========================================================
	//
	// Antes dos demais comandos verificamos se a mensagem
	// é simplesmente:
	//
	// A
	// B
	// C
	// D
	//
	// Caso exista um quiz ativo, a mensagem poderá ser
	// interpretada como resposta.
	//
	// ==========================================================

	if msgEvent.Info.IsGroup &&
		isQuizAnswer(text) {

		handled :=
			handleQuizAnswer(
				client,
				msgEvent,
				text,
			)

		if handled {
			logger.Info("==============================")
			return
		}
	}

	// ==========================================================
	// !beijo / !tapa
	// ==========================================================
	//
	// Comandos recreativos:
	//
	// !beijo @pessoa
	// !beijo
	// !tapa @pessoa
	// !tapa
	//
	// Quando não existe uma menção, o alvo é sorteado
	// aleatoriamente entre os participantes do grupo.
	//
	// ==========================================================

	// ==========================================================
	// !menu
	// ==========================================================

	// ==========================================================
	// !perfil
	// ==========================================================

	// ==========================================================
	// !slots
	// ==========================================================

	// ==========================================================
	// !duelo / !aceitar / !recusar
	// ==========================================================

	if handleDuelCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !caraoucoroa
	// ==========================================================

	// ==========================================================
	// !forca / !letra / !palavra
	// ==========================================================

	if handleForcaCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !loteria
	// ==========================================================

	if handleLotteryCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	if handleCoinflipCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	if handleSlotsCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !pressagio
	// ==========================================================

	if handleChaosCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !cofre
	// ==========================================================

	if handleVaultCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !conquistas
	// ==========================================================

	if handleAchievementsCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	if handleProfileCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	// ==========================================================
	// RPG
	// ==========================================================

	if handleRPGCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	if handleMenuCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	// ==========================================================
	// COMANDOS RECREATIVOS
	// ==========================================================

	// ==========================================================
	// !ship
	// ==========================================================

	if handleShipCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	if handleFunCommand(
		client,
		msgEvent,
		text,
	) {
		logger.Info("==============================")
		return
	}

	// ==========================================================
	// !gold
	// ==========================================================

	if command == "!gold" {
		if len(parts) != 1 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!gold*",
			)

			logger.Info("==============================")
			return
		}

		logger.Info(
			"Comando !gold recebido de:",
			canonicalSenderJID(msgEvent),
		)

		handleGoldStarterCommand(
			client,
			msgEvent,
		)

		logger.Info("==============================")
		return
	}

	// ==========================================================

	// ==========================================================
	// COMANDOS LEGADOS DE EQUIPAMENTO RPG
	// ==========================================================

	if command == "!espada" ||
		command == "!escudo" ||
		command == "!armadura" {

		if len(parts) != 1 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"Uso correto: *%s*",
					command,
				),
			)

			logger.Info("==============================")
			return
		}

		handleLegacyRPGEquipmentCommand(
			client,
			msgEvent,
			command,
		)

		logger.Info("==============================")
		return
	}

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

		playerState, err :=
			rpg.GetPlayer(
				groupJID,
				jid,
			)

		if err != nil {
			logger.Error(
				"Erro ao consultar Cristais para !saldo:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Erro ao consultar seus Cristais.",
			)

			logger.Info("==============================")
			return
		}

		response := fmt.Sprintf(
			"💰 *@%s*\n"+
				"💰 *%s*\n"+
				"💎 *%s Cristais*",
			name,
			formatGold(
				wallet.Gold,
			),
			formatRPGNumber(
				playerState.MagicCrystals,
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

		logger.Info(
			"Saldo consultado:",
			jid,
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
						"🍀 @%s, você já tentou sua sorte hoje!\n\n⏳ Próxima tentativa em: *%s*",
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

		progressText :=
			recordDailyLuckProgress(
				groupJID,
				jid,
			)

		headline :=
			luckHeadline(
				result.Tier,
			)

		response := fmt.Sprintf(
			"%s\n"+
				"%s *@%s* • *%s*\n"+
				"💰 +%s\n"+
				"Saldo: *%s*\n"+
				"⏳ 24h",
			headline,
			result.Emoji,
			name,
			result.Tier,
			formatGold(
				result.Amount,
			),
			formatGold(
				result.Balance,
			),
		)

		response += progressText

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
	// !quiz
	// ==========================================================

	if command == "!quiz" {
		if !requireGoldGroup(
			client,
			msgEvent,
			"!quiz",
		) {
			return
		}

		if len(parts) != 1 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"Uso correto: *!quiz*",
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

		// ======================================================
		// VERIFICAR CARTEIRA
		// ======================================================

		wallet, err :=
			database.GetWallet(
				groupJID,
				jid,
			)

		if err != nil {
			logger.Error(
				"Erro ao consultar carteira para quiz:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível iniciar o Quiz.",
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

		// ======================================================
		// INICIAR QUIZ
		// ======================================================

		session, err :=
			quiz.Start(
				groupJID,
				jid,
			)

		if err != nil {
			if errors.Is(
				err,
				quiz.ErrQuizActive,
			) {
				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"🧠 Existe um Quiz ativo no momento, aguarde sua vez.",
				)

				logger.Info("==============================")
				return
			}

			logger.Error(
				"Erro ao iniciar Quiz:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível iniciar o Quiz.",
			)

			logger.Info("==============================")
			return
		}

		// ======================================================
		// ENVIAR PERGUNTA
		// ======================================================

		response :=
			formatQuizQuestion(
				name,
				session,
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
			"Quiz iniciado:",
			"Grupo:",
			groupJID,
			"Usuário:",
			jid,
			"Dificuldade:",
			session.Difficulty,
			"Prêmio:",
			session.Reward,
		)

		// ======================================================
		// TIMER
		// ======================================================

		duration :=
			time.Until(
				session.ExpiresAt,
			)

		if duration < 0 {
			duration = 0
		}

		chatJID :=
			msgEvent.Info.Chat

		sessionID :=
			session.ID

		ownerJID :=
			jid

		ownerName :=
			name

		question :=
			session.Question

		reward :=
			session.Reward

		time.AfterFunc(
			duration,
			func() {
				expired :=
					quiz.Expire(
						groupJID,
						sessionID,
					)

				if !expired {
					return
				}

				correctLetter :=
					quizCorrectLetter(
						question.CorrectAnswer,
					)

				correctOption :=
					quizOptionByIndex(
						question,
						question.CorrectAnswer,
					)

				timeoutResponse := fmt.Sprintf(
					"⏰ *TEMPO ESGOTADO!*\n\n@%s não respondeu.\n✅ Correta: *%s)* %s\n💰 Prêmio perdido: *%s Gold*",
					ownerName,
					correctLetter,
					correctOption,
					formatRPGNumber(
						reward,
					),
				)

				parsedOwner,
					parseErr :=
					types.ParseJID(
						ownerJID,
					)

				if parseErr != nil {
					logger.Error(
						"Erro ao converter JID do jogador no timeout do Quiz:",
						parseErr,
					)

					_ = whatsapp.SendText(
						client,
						chatJID,
						timeoutResponse,
					)

					return
				}

				_ = whatsapp.SendMentionedText(
					client,
					chatJID,
					timeoutResponse,
					[]types.JID{
						parsedOwner.ToNonAD(),
					},
				)

				logger.Info(
					"Quiz expirado:",
					"Grupo:",
					groupJID,
					"Usuário:",
					ownerJID,
					"Prêmio perdido:",
					reward,
				)
			},
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
			"🏆 *MAIS RICOS*\n\n",
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
					"%s *%s* — %s\n",
					prefix,
					name,
					formatGold(
						entry.Gold,
					),
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
			"💸 *@%s → @%s*\n💰 %s\nSaldo: *%s*",
			senderName,
			targetName,
			formatGold(result.Amount),
			formatGold(result.SenderBalance),
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
				"🎲 *APOSTA*\n\n"+
					"Use:\n"+
					"*!bet <valor>*\n"+
					"*!bet <percentual>%*\n"+
					"*!bet metade*\n"+
					"*!bet all*\n\n"+
					"Exemplos:\n"+
					"*!bet 100000*\n"+
					"*!bet 20%*\n"+
					"*!bet metade*\n"+
					"*!bet all*",
			)

			logger.Info("==============================")
			return
		}

		groupJID :=
			msgEvent.Info.Chat.String()

		jid :=
			canonicalSenderJID(
				msgEvent,
			)

		currentBalance,
			balanceErr :=
			gold.GetBalance(
				groupJID,
				jid,
			)

		if balanceErr != nil {
			logger.Error(
				"Erro consultando saldo para !bet:",
				balanceErr,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível consultar seu saldo Gold.",
			)

			logger.Info("==============================")
			return
		}

		betAmount,
			err :=
			parseFlexibleGoldAmount(
				parts[1],
				currentBalance,
			)

		if err != nil {
			switch {
			case errors.Is(
				err,
				errFlexibleAmountPercentRange,
			):
				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ O percentual da aposta precisa estar entre *1% e 100%*.\n\n"+
						"Exemplo: *!bet 20%*",
				)

			case errors.Is(
				err,
				errFlexibleAmountNoBalance,
			):
				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ Você não possui Gold disponível para essa aposta.\n\n"+
						"Use *!saldo* para consultar sua carteira.",
				)

			default:
				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ Valor de aposta inválido.\n\n"+
						"Use, por exemplo:\n"+
						"*!bet 100000*\n"+
						"*!bet 20%*\n"+
						"*!bet metade*\n"+
						"*!bet all*",
				)
			}

			logger.Info("==============================")
			return
		}

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

		progressText :=
			recordBetProgress(
				groupJID,
				jid,
				result.Multiplier,
			)

		var response string

		if result.NetResult < 0 {
			response = fmt.Sprintf(
				"💀 *Perdeu*\n"+
					"💸 %s\n"+
					"💰 *%s*",
				formatSignedGold(
					result.NetResult,
				),
				formatGold(
					result.Balance,
				),
			)

		} else if result.NetResult == 0 {
			response = fmt.Sprintf(
				"😐 *Empate*\n"+
					"↩️ %s\n"+
					"💰 *%s*",
				formatGold(
					result.Prize,
				),
				formatGold(
					result.Balance,
				),
			)

		} else {
			response = fmt.Sprintf(
				"🎉 *%s*\n"+
					"🏆 %s\n"+
					"💰 *%s*",
				result.Result,
				formatSignedGold(
					result.NetResult,
				),
				formatGold(
					result.Balance,
				),
			)
		}

		response = fmt.Sprintf(
			"@%s\n\n%s",
			name,
			response,
		)

		response += progressText

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

		combat, err :=
			prepareRobberyCombat(
				groupJID,
				jid,
				targetJID,
			)

		if err != nil {
			switch {
			case errors.Is(
				err,
				errRobberyRobberWalletMissing,
			):
				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					fmt.Sprintf(
						"❌ Você ainda não possui uma carteira Gold neste grupo.\\n\\nUse *!gold* para receber seus *%d Gold* iniciais e iniciar sua jornada RPG.",
						gold.InitialGold,
					),
				)

			case errors.Is(
				err,
				errRobberyTargetWalletMissing,
			):
				response := fmt.Sprintf(
					"💸 *@%s* ainda não possui carteira Gold neste grupo e não pode ser roubado.",
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

			default:
				logger.Error(
					"Erro ao preparar combate RPG do roubo:",
					err,
				)

				_ = whatsapp.SendText(
					client,
					msgEvent.Info.Chat,
					"❌ Não foi possível preparar o combate RPG.",
				)
			}

			logger.Info("==============================")
			return
		}

		logger.Info(
			"Combate RPG do !roubar:",
			"Ladrão PC:",
			combat.RobberPower,
			"Alvo PC:",
			combat.TargetPower,
			"Chance:",
			combat.SuccessChance,
			"%",
		)

		result, err :=
			gold.Rob(
				groupJID,
				jid,
				targetJID,
				combat.SuccessChance,
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

		progressText :=
			recordRobberyProgress(
				groupJID,
				jid,
				result.Success,
			)

		var response string

		if result.Success {
			response = fmt.Sprintf(
				"🦹 *@%s roubou @%s*\n"+
					"💰 +%s\n"+
					"💼 *%s*",
				robberName,
				targetName,
				formatGold(
					result.Amount,
				),
				formatGold(
					result.RobberBalance,
				),
			)
		} else {
			response = fmt.Sprintf(
				"🚨 *@%s falhou contra @%s*\n"+
					"💸 -%s\n"+
					"💼 *%s*",
				robberName,
				targetName,
				formatGold(
					result.Penalty,
				),
				formatGold(
					result.RobberBalance,
				),
			)
		}

		response += progressText

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
		// ======================================================
		// COMANDO DESCONHECIDO
		// ======================================================
		//
		// Neste ponto todos os comandos válidos já foram
		// processados. Se a mensagem começar com "!", ela
		// representa uma tentativa de utilizar um comando
		// inexistente.
		//
		// Mensagens comuns continuam sendo ignoradas.
		//

		if handleUnknownCommand(
			client,
			msgEvent,
			text,
		) {
			logger.Info("==============================")
			return
		}

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
// QUIZ - RESPOSTAS
// ==========================================================

func isQuizAnswer(
	text string,
) bool {

	normalized :=
		strings.TrimSpace(
			strings.ToUpper(
				text,
			),
		)

	switch normalized {
	case "A", "B", "C", "D":
		return true

	default:
		return false
	}
}

// handleQuizAnswer retorna true quando a mensagem fazia parte
// de um Quiz ativo.
//
// Respostas de pessoas que não iniciaram a rodada são
// silenciosamente ignoradas.
func handleQuizAnswer(
	client *whatsmeow.Client,
	msg *events.Message,
	answer string,
) bool {

	groupJID :=
		msg.Info.Chat.String()

	session, exists :=
		quiz.GetActive(
			groupJID,
		)

	if !exists {
		return false
	}

	jid :=
		canonicalSenderJID(msg)

	// Existe Quiz, mas a mensagem veio de outra pessoa.
	//
	// Ignoramos para não poluir o grupo.
	if session.OwnerJID != jid {
		logger.Debug(
			"Resposta de Quiz ignorada de usuário que não iniciou a rodada:",
			jid,
		)

		return true
	}

	result, err :=
		quiz.Answer(
			groupJID,
			jid,
			answer,
		)

	if err != nil {
		switch {
		case errors.Is(
			err,
			quiz.ErrQuizExpired,
		):
			return true

		case errors.Is(
			err,
			quiz.ErrNoActiveQuiz,
		):
			return true

		case errors.Is(
			err,
			quiz.ErrNotQuizOwner,
		):
			return true

		case errors.Is(
			err,
			quiz.ErrInvalidAnswer,
		):
			return true

		default:
			logger.Error(
				"Erro ao processar resposta do Quiz:",
				err,
			)

			return true
		}
	}

	name :=
		msg.Info.PushName

	selectedIndex :=
		quizIndexFromLetter(
			result.SelectedAnswer,
		)

	selectedOption :=
		quizOptionByIndex(
			result.Question,
			selectedIndex,
		)

	correctOption :=
		quizOptionByIndex(
			result.Question,
			result.Question.CorrectAnswer,
		)

	// ======================================================
	// ACERTO
	// ======================================================

	if result.Correct {
		quizReward,
			quizChaosBonus,
			quizChaosPercent :=
			applyChaosGameReward(
				result.Reward,
			)

		rewardResult, rewardErr :=
			gold.RewardQuiz(
				groupJID,
				jid,
				quizReward,
				string(
					result.Difficulty,
				),
			)

		if rewardErr != nil {
			logger.Error(
				"Erro ao pagar prêmio do Quiz:",
				rewardErr,
			)

			_ = whatsapp.SendText(
				client,
				msg.Info.Chat,
				"❌ Você acertou o Quiz, mas ocorreu um erro ao creditar o prêmio.",
			)

			return true
		}

		progressText :=
			recordQuizProgress(
				groupJID,
				jid,
				result.Difficulty,
				true,
			)

		response := fmt.Sprintf(
			"✅ *RESPOSTA CORRETA!*\n\n@%s • *%s)* %s\n💰 *+%s Gold* • Saldo: *%s Gold*",
			name,
			result.CorrectAnswer,
			correctOption,
			formatRPGNumber(
				rewardResult.Amount,
			),
			formatRPGNumber(
				rewardResult.Balance,
			),
		)

		if quizChaosBonus > 0 {
			response +=
				formatChaosGoldBonus(
					quizChaosBonus,
					quizChaosPercent,
				)
		}

		response += progressText

		_ = whatsapp.SendMentionedText(
			client,
			msg.Info.Chat,
			response,
			[]types.JID{
				msg.Info.Sender.ToNonAD(),
			},
		)

		logger.Success(
			"Quiz respondido corretamente:",
			"Usuário:",
			jid,
			"Dificuldade:",
			result.Difficulty,
			"Prêmio:",
			rewardResult.Amount,
		)

		return true
	}

	// ======================================================
	// ERRO
	// ======================================================

	_ = recordQuizProgress(
		groupJID,
		jid,
		result.Difficulty,
		false,
	)

	response := fmt.Sprintf(
		"❌ *RESPOSTA ERRADA!*\n\n@%s • marcou *%s)* %s\n✅ Correta: *%s)* %s\n💰 Prêmio perdido: *%s Gold*",
		name,
		result.SelectedAnswer,
		selectedOption,
		result.CorrectAnswer,
		correctOption,
		formatRPGNumber(
			result.Reward,
		),
	)

	_ = whatsapp.SendMentionedText(
		client,
		msg.Info.Chat,
		response,
		[]types.JID{
			msg.Info.Sender.ToNonAD(),
		},
	)

	logger.Info(
		"Quiz respondido incorretamente:",
		"Usuário:",
		jid,
		"Dificuldade:",
		result.Difficulty,
		"Resposta:",
		result.SelectedAnswer,
		"Correta:",
		result.CorrectAnswer,
	)

	return true
}

// ==========================================================
// QUIZ - FORMATAÇÃO
// ==========================================================

func formatQuizQuestion(
	name string,
	session *quiz.Session,
) string {

	seconds :=
		quizTimeLimitSeconds(
			session.Difficulty,
		)

	title :=
		fmt.Sprintf(
			"🧠 *QUIZ • %s*",
			session.Difficulty,
		)

	if session.Difficulty ==
		quiz.DifficultyInsane {

		title =
			"🔥 *QUIZ INSANO* 🔥"
	}

	response := fmt.Sprintf(
		"%s\n\n@%s\n*%s*\n\n*A)* %s\n*B)* %s\n*C)* %s\n*D)* %s\n\n💰 *%s Gold* • ⏳ *%ds*\n📝 Responda com *A, B, C ou D*.",
		title,
		name,
		session.Question.Text,
		session.Question.Options[0],
		session.Question.Options[1],
		session.Question.Options[2],
		session.Question.Options[3],
		formatRPGNumber(session.Reward),
		seconds,
	)

	if session.Difficulty ==
		quiz.DifficultyInsane {

		response = fmt.Sprintf(
			"%s\n\n@%s\n*%s*\n\n*A)* %s\n*B)* %s\n*C)* %s\n*D)* %s\n\n👑 *%s Gold* • ⏳ *%ds*\n📝 Responda com *A, B, C ou D*.",
			title,
			name,
			session.Question.Text,
			session.Question.Options[0],
			session.Question.Options[1],
			session.Question.Options[2],
			session.Question.Options[3],
			formatRPGNumber(session.Reward),
			seconds,
		)
	}

	return response
}

func quizTimeLimitSeconds(
	difficulty quiz.Difficulty,
) int {

	if difficulty ==
		quiz.DifficultyInsane {

		return int(
			quiz.InsaneTimeLimit.Seconds(),
		)
	}

	return int(
		quiz.NormalTimeLimit.Seconds(),
	)
}

func quizCorrectLetter(
	index int,
) string {

	switch index {
	case 0:
		return "A"

	case 1:
		return "B"

	case 2:
		return "C"

	case 3:
		return "D"

	default:
		return "?"
	}
}

func quizIndexFromLetter(
	letter string,
) int {

	switch strings.ToUpper(
		strings.TrimSpace(
			letter,
		),
	) {
	case "A":
		return 0

	case "B":
		return 1

	case "C":
		return 2

	case "D":
		return 3

	default:
		return -1
	}
}

func quizOptionByIndex(
	question quiz.Question,
	index int,
) string {

	if index < 0 ||
		index >= len(question.Options) {

		return "Resposta desconhecida"
	}

	return question.Options[index]
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

func formatBetPayout(
	percent int,
) string {
	if percent <= 0 {
		return "0x"
	}

	whole :=
		percent / 100

	decimal :=
		percent % 100

	if decimal == 0 {
		return fmt.Sprintf(
			"%dx",
			whole,
		)
	}

	if decimal%10 == 0 {
		return fmt.Sprintf(
			"%d,%dx",
			whole,
			decimal/10,
		)
	}

	return fmt.Sprintf(
		"%d,%02dx",
		whole,
		decimal,
	)
}

func rankingDisplayName(
	name string,
	jid string,
) string {
	name =
		database.NormalizeUserDisplayName(
			jid,
			name,
		)

	if name != "" {
		return name
	}

	return "Aventureiro"
}

// ==========================================================
// FORMATAÇÃO DE TEMPO
// ==========================================================

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
