package handler

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"whatsapp-sticker-bot/internal/duel"
	"whatsapp-sticker-bot/internal/gold"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/profile"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var duelBattleMessages = []string{
	"💥 @%s acertou um golpe crítico e derrubou @%s!",
	"⚡ @%s foi mais rápido e derrotou @%s em um ataque certeiro!",
	"🗡️ @%s encontrou uma abertura na defesa de @%s e venceu o duelo!",
	"🔥 @%s partiu para cima e não deu chance para @%s reagir!",
	"💫 @%s desviou no último segundo e contra-atacou @%s com precisão!",
	"🥊 @%s acertou uma sequência perfeita e venceu @%s!",
	"⚔️ Depois de uma batalha intensa, @%s levou a melhor sobre @%s!",
	"🎯 @%s esperou o momento certo e finalizou o duelo contra @%s!",
	"🛡️ @%s resistiu aos ataques e virou a luta contra @%s!",
	"🏹 @%s executou o ataque decisivo e derrotou @%s!",
}

func handleDuelCommand(
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
		strings.ToLower(parts[0])

	switch command {
	case "!duelo":
		handleDuelChallenge(
			client,
			msgEvent,
			parts,
		)
		return true

	case "!aceitar":
		handleDuelAccept(
			client,
			msgEvent,
			parts,
		)
		return true

	case "!recusar":
		handleDuelReject(
			client,
			msgEvent,
			parts,
		)
		return true

	default:
		return false
	}
}

func handleDuelChallenge(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {
	if !requireGoldGroup(
		client,
		msgEvent,
		"!duelo",
	) {
		return
	}

	if len(parts) < 3 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"⚔️ Uso correto: *!duelo @pessoa <valor>*\n\nExemplo: *!duelo @pessoa 1000*\n\n💰 Aposta mínima: *%d Gold*",
				gold.DuelMinBet,
			),
		)

		return
	}

	targetJID, ok :=
		getSingleMention(
			msgEvent,
		)

	if !ok {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Você precisa mencionar exatamente uma pessoa para desafiar.",
		)

		return
	}

	amount, err :=
		strconv.Atoi(
			parts[len(parts)-1],
		)

	if err != nil ||
		amount < gold.DuelMinBet {

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"❌ A aposta do duelo precisa ser um número inteiro de pelo menos *%d Gold*.",
				gold.DuelMinBet,
			),
		)

		return
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	challengerJID :=
		canonicalSenderJID(
			msgEvent,
		)

	groupInfo, err :=
		client.GetGroupInfo(
			context.Background(),
			msgEvent.Info.Chat,
		)

	if err != nil {
		logger.Error(
			"Erro ao buscar participantes para !duelo:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível verificar o participante agora.",
		)

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

		return
	}

	if targetJID == challengerJID {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"😂 Você não pode desafiar a si mesmo para um duelo.",
		)

		return
	}

	if err :=
		gold.ValidateDuel(
			groupJID,
			challengerJID,
			targetJID,
			amount,
		); err != nil {

		sendDuelValidationError(
			client,
			msgEvent,
			err,
		)

		return
	}

	challengerName :=
		strings.TrimSpace(
			msgEvent.Info.PushName,
		)

	if challengerName == "" {
		challengerName =
			rankingDisplayName(
				"",
				challengerJID,
			)
	}

	targetName :=
		pixTargetName(parts)

	challenge :=
		duel.Challenge{
			GroupJID: groupJID,

			ChallengerJID: challengerJID,

			ChallengerName: challengerName,

			TargetJID: targetJID,

			TargetName: targetName,

			Amount: amount,
		}

	if err :=
		duel.Create(
			challenge,
		); err != nil {

		if errors.Is(
			err,
			duel.ErrChallengeExists,
		) {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"⚔️ Já existe um duelo pendente neste grupo. Aguarde ele ser aceito, recusado ou expirar.",
			)

			return
		}

		logger.Error(
			"Erro ao criar desafio de duelo:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível criar o duelo agora.",
		)

		return
	}

	targetMention, err :=
		types.ParseJID(
			targetJID,
		)

	if err != nil {
		logger.Error(
			"Erro ao converter JID do desafiado:",
			err,
		)

		return
	}

	response := fmt.Sprintf(
		"⚔️ *DESAFIO DE DUELO*\n\n@%s desafiou @%s!\n\n💰 Aposta de cada jogador: *%d Gold*\n🏆 Pote total: *%d Gold*\n\n@%s, responda com:\n*!aceitar*\nou\n*!recusar*\n\n⏳ O desafio expira em *60 segundos*.",
		challengerName,
		targetName,
		amount,
		amount*2,
		targetName,
	)

	_ = whatsapp.SendMentionedText(
		client,
		msgEvent.Info.Chat,
		response,
		[]types.JID{
			msgEvent.Info.
				Sender.
				ToNonAD(),

			targetMention.
				ToNonAD(),
		},
	)

	logger.Info(
		"Duelo criado:",
		challengerJID,
		"vs",
		targetJID,
		"Aposta:",
		amount,
	)
}

func handleDuelAccept(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {
	if !requireGoldGroup(
		client,
		msgEvent,
		"!aceitar",
	) {
		return
	}

	if len(parts) != 1 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"Uso correto: *!aceitar*",
		)

		return
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	responderJID :=
		canonicalSenderJID(
			msgEvent,
		)

	challenge, err :=
		duel.Accept(
			groupJID,
			responderJID,
		)

	if err != nil {
		sendDuelSessionError(
			client,
			msgEvent,
			err,
		)

		return
	}

	result, err :=
		gold.ResolveDuel(
			groupJID,
			challenge.ChallengerJID,
			challenge.TargetJID,
			challenge.Amount,
		)

	if err != nil {
		sendDuelResolutionError(
			client,
			msgEvent,
			err,
		)

		return
	}

	winnerName :=
		challenge.ChallengerName

	loserName :=
		challenge.TargetName

	if result.WinnerJID ==
		challenge.TargetJID {

		winnerName =
			challenge.TargetName

		loserName =
			challenge.ChallengerName
	}

	winnerProgress :=
		recordDuelProgress(
			groupJID,
			result.WinnerJID,
			winnerName,
			true,
		)

	loserProgress :=
		recordDuelProgress(
			groupJID,
			result.LoserJID,
			loserName,
			false,
		)

	challengerMention, challengerErr :=
		types.ParseJID(
			challenge.ChallengerJID,
		)

	targetMention, targetErr :=
		types.ParseJID(
			challenge.TargetJID,
		)

	if challengerErr != nil ||
		targetErr != nil {

		logger.Error(
			"Erro ao converter participantes do duelo:",
			challengerErr,
			targetErr,
		)

		return
	}

	battleText :=
		randomDuelBattleMessage(
			winnerName,
			loserName,
		)

	response := fmt.Sprintf(
		"⚔️ *DUELO FINALIZADO!*\n\n@%s ⚔️ @%s\n\n%s\n\n🏆 Vencedor: *@%s*\n💰 Pote: *%d Gold*\n📈 Lucro líquido: *+%d Gold*\n\n💰 Saldo do vencedor: *%d Gold*\n💸 Saldo do perdedor: *%d Gold*%s%s",
		challenge.ChallengerName,
		challenge.TargetName,
		battleText,
		winnerName,
		result.Pot,
		result.BetAmount,
		result.WinnerBalance,
		result.LoserBalance,
		winnerProgress,
		loserProgress,
	)

	_ = whatsapp.SendMentionedText(
		client,
		msgEvent.Info.Chat,
		response,
		[]types.JID{
			challengerMention.
				ToNonAD(),

			targetMention.
				ToNonAD(),
		},
	)

	logger.Info(
		"Duelo finalizado:",
		challenge.ChallengerJID,
		"vs",
		challenge.TargetJID,
		"Vencedor:",
		result.WinnerJID,
	)
}

func handleDuelReject(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {
	if !requireGoldGroup(
		client,
		msgEvent,
		"!recusar",
	) {
		return
	}

	if len(parts) != 1 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"Uso correto: *!recusar*",
		)

		return
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	responderJID :=
		canonicalSenderJID(
			msgEvent,
		)

	challenge, err :=
		duel.Reject(
			groupJID,
			responderJID,
		)

	if err != nil {
		sendDuelSessionError(
			client,
			msgEvent,
			err,
		)

		return
	}

	challengerMention, challengerErr :=
		types.ParseJID(
			challenge.ChallengerJID,
		)

	targetMention, targetErr :=
		types.ParseJID(
			challenge.TargetJID,
		)

	if challengerErr != nil ||
		targetErr != nil {

		logger.Error(
			"Erro ao converter participantes do duelo recusado:",
			challengerErr,
			targetErr,
		)

		return
	}

	response := fmt.Sprintf(
		"🕊️ *DUELO RECUSADO*\n\n@%s recusou o desafio de @%s.\n\n💰 Nenhum Gold foi movimentado.",
		challenge.TargetName,
		challenge.ChallengerName,
	)

	_ = whatsapp.SendMentionedText(
		client,
		msgEvent.Info.Chat,
		response,
		[]types.JID{
			challengerMention.
				ToNonAD(),

			targetMention.
				ToNonAD(),
		},
	)
}

func sendDuelValidationError(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	err error,
) {
	var response string

	switch {
	case errors.Is(
		err,
		gold.ErrDuelChallengerWalletNotFound,
	):
		response =
			"❌ Você precisa criar sua carteira neste grupo usando *!gold* antes de desafiar alguém."

	case errors.Is(
		err,
		gold.ErrDuelTargetWalletNotFound,
	):
		response =
			"❌ A pessoa desafiada ainda não possui uma carteira Gold neste grupo."

	case errors.Is(
		err,
		gold.ErrDuelChallengerInsufficientGold,
	):
		response =
			"❌ Você não possui Gold suficiente para essa aposta."

	case errors.Is(
		err,
		gold.ErrDuelTargetInsufficientGold,
	):
		response =
			"❌ A pessoa desafiada não possui Gold suficiente para essa aposta."

	case errors.Is(
		err,
		gold.ErrDuelSameUser,
	):
		response =
			"😂 Você não pode desafiar a si mesmo."

	default:
		logger.Error(
			"Erro ao validar duelo:",
			err,
		)

		response =
			"❌ Não foi possível validar o duelo agora."
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}

func sendDuelSessionError(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	err error,
) {
	var response string

	switch {
	case errors.Is(
		err,
		duel.ErrChallengeMissing,
	):
		response =
			"⚔️ Não existe nenhum duelo pendente neste grupo."

	case errors.Is(
		err,
		duel.ErrChallengeExpired,
	):
		response =
			"⌛ O desafio de duelo expirou. Um novo desafio precisa ser criado."

	case errors.Is(
		err,
		duel.ErrNotTarget,
	):
		response =
			"⚔️ Apenas a pessoa que foi desafiada pode aceitar ou recusar este duelo."

	default:
		logger.Error(
			"Erro na sessão de duelo:",
			err,
		)

		response =
			"❌ Não foi possível processar o duelo agora."
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}

func sendDuelResolutionError(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	err error,
) {
	var response string

	switch {
	case errors.Is(
		err,
		gold.ErrDuelChallengerInsufficientGold,
	):
		response =
			"❌ O duelo foi cancelado porque o desafiante não possui mais Gold suficiente para a aposta."

	case errors.Is(
		err,
		gold.ErrDuelTargetInsufficientGold,
	):
		response =
			"❌ O duelo foi cancelado porque o desafiado não possui mais Gold suficiente para a aposta."

	case errors.Is(
		err,
		gold.ErrDuelChallengerWalletNotFound,
	),
		errors.Is(
			err,
			gold.ErrDuelTargetWalletNotFound,
		):

		response =
			"❌ O duelo foi cancelado porque uma das carteiras não existe mais."

	default:
		logger.Error(
			"Erro ao resolver duelo:",
			err,
		)

		response =
			"❌ O duelo não pôde ser concluído. Nenhum resultado foi aplicado."
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}

func recordDuelProgress(
	groupJID string,
	jid string,
	name string,
	won bool,
) string {
	result, err :=
		profile.RecordDuelResult(
			groupJID,
			jid,
			won,
		)

	if err != nil {
		logger.Error(
			"Erro ao registrar progresso do duelo:",
			err,
		)

		return ""
	}

	if result == nil ||
		result.Gained <= 0 {

		return ""
	}

	if result.After.Level >
		result.Before.Level {

		return fmt.Sprintf(
			"\n✨ @%s: *+%d XP* • 🎉 *LEVEL UP %d → %d!*",
			name,
			result.Gained,
			result.Before.Level,
			result.After.Level,
		)
	}

	return fmt.Sprintf(
		"\n✨ @%s: *+%d XP*",
		name,
		result.Gained,
	)
}

func randomDuelBattleMessage(
	winnerName string,
	loserName string,
) string {
	if len(duelBattleMessages) == 0 {
		return fmt.Sprintf(
			"⚔️ @%s derrotou @%s!",
			winnerName,
			loserName,
		)
	}

	n, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(
				int64(
					len(
						duelBattleMessages,
					),
				),
			),
		)

	if err != nil {
		return fmt.Sprintf(
			"⚔️ @%s derrotou @%s!",
			winnerName,
			loserName,
		)
	}

	message := duelBattleMessages[int(n.Int64())]

	return fmt.Sprintf(
		message,
		winnerName,
		loserName,
	)
}
