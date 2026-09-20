package handler

import (
	"context"
	"fmt"
	"strings"

	"whatsapp-sticker-bot/internal/database"
	"whatsapp-sticker-bot/internal/gold"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/profile"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// handleProfileCommand processa:
//
// !perfil
// !perfil @pessoa
//
// O perfil é sempre específico do grupo atual.
func handleProfileCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts := strings.Fields(text)

	if len(parts) == 0 ||
		!strings.EqualFold(
			parts[0],
			"!perfil",
		) {

		return false
	}

	if !requireGoldGroup(
		client,
		msgEvent,
		"!perfil",
	) {
		return true
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	targetJID :=
		canonicalSenderJID(
			msgEvent,
		)

	mentionJID :=
		msgEvent.Info.
			Sender.
			ToNonAD()

	viewingSelf := true

	// Se houver conteúdo depois de !perfil,
	// exigimos uma pessoa mencionada.
	if len(parts) > 1 {
		mentionedJID, ok :=
			getSingleMention(
				msgEvent,
			)

		if !ok {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"👤 Use *!perfil* ou *!perfil @pessoa*.",
			)

			return true
		}

		groupInfo, err :=
			client.GetGroupInfo(
				context.Background(),
				msgEvent.Info.Chat,
			)

		if err != nil {
			logger.Error(
				"Erro ao consultar grupo para !perfil:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível consultar esse perfil agora.",
			)

			return true
		}

		resolvedJID, ok :=
			resolveParticipantJID(
				groupInfo.Participants,
				mentionedJID,
			)

		if !ok {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ A pessoa mencionada não pertence a este grupo.",
			)

			return true
		}

		targetJID = resolvedJID

		viewingSelf =
			targetJID ==
				canonicalSenderJID(
					msgEvent,
				)

		parsedJID, err :=
			types.ParseJID(
				targetJID,
			)

		if err != nil {
			logger.Error(
				"Erro ao converter JID do perfil:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível identificar esse jogador.",
			)

			return true
		}

		mentionJID =
			parsedJID.ToNonAD()
	}

	wallet, err :=
		database.GetWallet(
			groupJID,
			targetJID,
		)

	if err != nil {
		logger.Error(
			"Erro ao consultar carteira para !perfil:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível consultar o perfil agora.",
		)

		return true
	}

	if wallet == nil {
		var response string

		if viewingSelf {
			response = fmt.Sprintf(
				"❌ Você ainda não possui uma carteira Gold neste grupo.\n\nUse *!gold* para receber seus *%d Gold* iniciais e criar seu perfil.",
				gold.InitialGold,
			)
		} else {
			response =
				"❌ Essa pessoa ainda não possui uma carteira Gold neste grupo."
		}

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			response,
		)

		return true
	}

	player, err :=
		profile.Get(
			groupJID,
			targetJID,
		)

	if err != nil {
		logger.Error(
			"Erro ao carregar perfil:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível carregar o perfil agora.",
		)

		return true
	}

	if player == nil {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Perfil não encontrado.",
		)

		return true
	}

	name :=
		strings.TrimSpace(
			player.Name,
		)

	if name == "" {
		name =
			strings.TrimSpace(
				mentionJID.User,
			)
	}

	if name == "" {
		name = "jogador"
	}

	level :=
		profile.LevelFromXP(
			player.XP,
		)

	progressBar :=
		profileProgressBar(
			level.ProgressPercent,
		)

	coinflipStats, err :=
		profile.GetCoinflipStats(
			groupJID,
			targetJID,
		)

	if err != nil {
		logger.Error(
			"Erro ao consultar estatísticas do Cara ou Coroa:",
			err,
		)

		coinflipStats =
			&profile.CoinflipStats{}
	}

	lotteryStats, err :=
		profile.GetLotteryStats(
			groupJID,
			targetJID,
		)

	if err != nil {
		logger.Error(
			"Erro ao consultar estatísticas da Loteria:",
			err,
		)

		lotteryStats =
			&profile.LotteryStats{}
	}

	response := fmt.Sprintf(
		"👤 *PERFIL DO JOGADOR*\n\n"+
			"@%s\n\n"+
			"⭐ Nível: *%d*\n"+
			"✨ XP total: *%d*\n"+
			"📈 Próximo nível: *%d / %d XP*\n"+
			"%s *%d%%*\n\n"+
			"💰 Gold: *%d*\n"+
			"🏆 Ranking Gold: *#%d de %d*\n\n"+
			"📊 *ESTATÍSTICAS*\n\n"+
			"🧠 Quiz: *%d* acertos • *%d* erros\n"+
			"💀 Quiz Insano: *%d* acertos\n"+
			"🎲 Apostas: *%d* vitórias / *%d* partidas\n"+
			"🎰 Slots: *%d* vitórias / *%d* partidas\n"+
			"💎 Maior prêmio no Slots: *%d Gold*\n"+
			"🪙 Cara ou Coroa: *%d* vitórias / *%d* partidas\n"+
			"🎟️ Loteria: *%d* vitórias • *%d* bilhetes\n"+
			"🏆 Gold ganho na Loteria: *%d Gold*\n"+
			"⚔️ Duelos: *%d* vitórias • *%d* derrotas\n"+
			"🦹 Roubos: *%d* sucessos • *%d* falhas",
		name,
		level.Level,
		level.TotalXP,
		level.CurrentLevelXP,
		level.RequiredLevelXP,
		progressBar,
		level.ProgressPercent,
		player.Gold,
		player.GoldRank,
		player.Players,
		player.QuizCorrect,
		player.QuizWrong,
		player.QuizInsaneWins,
		player.BetsWon,
		player.BetsPlayed,
		player.SlotsWon,
		player.SlotsPlayed,
		player.SlotsBiggestPrize,
		coinflipStats.Won,
		coinflipStats.Played,
		lotteryStats.Wins,
		lotteryStats.TicketsBought,
		lotteryStats.GoldWon,
		player.DuelsWon,
		player.DuelsLost,
		player.RobberiesSuccess,
		player.RobberiesFailed,
	)

	err =
		whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{
				mentionJID,
			},
		)

	if err != nil {
		logger.Error(
			"Erro ao enviar !perfil:",
			err,
		)

		return true
	}

	logger.Info(
		"Perfil consultado:",
		targetJID,
		"Grupo:",
		groupJID,
	)

	return true
}

func profileProgressBar(
	percent int,
) string {
	if percent < 0 {
		percent = 0
	}

	if percent > 100 {
		percent = 100
	}

	filled :=
		percent / 10

	empty :=
		10 - filled

	return strings.Repeat(
		"█",
		filled,
	) +
		strings.Repeat(
			"░",
			empty,
		)
}
