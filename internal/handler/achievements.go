package handler

import (
	"context"
	"fmt"
	"strings"

	"whatsapp-sticker-bot/internal/database"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/profile"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func handleAchievementsCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts := strings.Fields(text)

	if len(parts) == 0 ||
		!strings.EqualFold(parts[0], "!conquistas") {
		return false
	}

	if !msgEvent.Info.IsGroup {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ O comando *!conquistas* só pode ser usado em grupos.",
		)
		return true
	}

	mentioned := funMentionedJIDs(msgEvent)

	if len(mentioned) > 1 ||
		(len(mentioned) == 0 && len(parts) != 1) {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"🏆 Use *!conquistas* ou *!conquistas @pessoa*.",
		)
		return true
	}

	groupJID := msgEvent.Info.Chat.String()
	targetJID := canonicalSenderJID(msgEvent)
	targetMention := msgEvent.Info.Sender.ToNonAD()
	targetName := strings.TrimSpace(msgEvent.Info.PushName)

	if targetName == "" {
		targetName = targetMention.User
	}

	if len(mentioned) == 1 {
		groupInfo, err := client.GetGroupInfo(
			context.Background(),
			msgEvent.Info.Chat,
		)
		if err != nil {
			logger.Error(
				"Erro ao consultar participantes para !conquistas:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível consultar os participantes do grupo agora.",
			)

			return true
		}

		resolvedJID, ok := resolveParticipantJID(
			groupInfo.Participants,
			mentioned[0],
		)
		if !ok {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ A pessoa mencionada não pertence a este grupo.",
			)

			return true
		}

		parsedJID, err := types.ParseJID(
			resolvedJID,
		)
		if err != nil {
			logger.Error(
				"Erro ao interpretar participante em !conquistas:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível identificar a pessoa mencionada.",
			)

			return true
		}

		targetMention =
			parsedJID.ToNonAD()

		targetJID =
			targetMention.String()

		targetName =
			funMentionLabel(
				text,
				"!conquistas",
			)

		if targetName == "" {
			targetName =
				targetMention.User
		}
	}

	exists, err :=
		achievementWalletExists(
			groupJID,
			targetJID,
		)

	if err != nil {
		logger.Error(
			"Erro ao verificar carteira para !conquistas:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível consultar as conquistas agora.",
		)

		return true
	}

	if !exists {
		if len(mentioned) == 0 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"🏆 Você ainda não possui um perfil neste grupo. Use *!gold* primeiro.",
			)
		} else {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"🏆 Essa pessoa ainda não possui um perfil neste grupo.",
			)
		}

		return true
	}

	player, err :=
		profile.Get(
			groupJID,
			targetJID,
		)

	if err != nil {
		logger.Error(
			"Erro ao consultar perfil para !conquistas:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível consultar as conquistas agora.",
		)

		return true
	}

	if player == nil {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"🏆 Nenhum perfil encontrado para este jogador.",
		)

		return true
	}

	achievements :=
		profile.EvaluateAchievements(
			player,
		)

	unlocked :=
		profile.CountUnlockedAchievements(
			achievements,
		)

	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"🏆 *CONQUISTAS DE @%s*\n\n"+
			"✅ *%d/%d desbloqueadas*\n\n",
		targetName,
		unlocked,
		len(achievements),
	)

	for _, achievement := range achievements {

		if achievement.Unlocked {
			fmt.Fprintf(
				&builder,
				"✅ %s *%s*\n   %s\n",
				achievement.Icon,
				achievement.Name,
				achievement.Description,
			)

			continue
		}

		progress :=
			achievement.Progress

		if progress >
			achievement.Target {

			progress =
				achievement.Target
		}

		fmt.Fprintf(
			&builder,
			"🔒 %s *%s* — %d/%d\n",
			achievement.Icon,
			achievement.Name,
			progress,
			achievement.Target,
		)
	}

	err =
		whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			strings.TrimSpace(
				builder.String(),
			),
			[]types.JID{
				targetMention,
			},
		)

	if err != nil {
		logger.Error(
			"Erro ao enviar !conquistas:",
			err,
		)
	}

	return true
}

func achievementWalletExists(
	groupJID string,
	jid string,
) (bool, error) {
	var count int

	err := database.DB.QueryRow(`
		SELECT COUNT(*)
		FROM group_wallets
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	).Scan(
		&count,
	)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
