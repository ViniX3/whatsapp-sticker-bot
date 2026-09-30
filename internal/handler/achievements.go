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

	mode := "active"
	invalidArgument := false

	for _, raw := range parts[1:] {
		arg :=
			strings.ToLower(
				strings.TrimSpace(
					raw,
				),
			)

		if arg == "" ||
			strings.HasPrefix(
				arg,
				"@",
			) {

			continue
		}

		switch arg {

		case "ativas",
			"ativos",
			"andamento":

			mode = "active"

		case "finalizadas",
			"terminadas",
			"concluidas",
			"concluídas":

			mode = "finished"

		case "bloqueadas",
			"bloqueados":

			mode = "locked"

		default:
			invalidArgument = true
		}
	}

	if len(mentioned) > 1 ||
		invalidArgument {

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"🏆 *CONQUISTAS*\n"+
				"*!conquistas*\n"+
				"*!conquistas finalizadas*\n"+
				"*!conquistas bloqueadas*",
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

	var builder strings.Builder

	switch mode {

	case "finished":
		fmt.Fprintf(
			&builder,
			"✅ *FINALIZADAS DE @%s*\n\n",
			targetName,
		)

		count := 0

		for _, achievement := range achievements {

			if !achievement.Unlocked {
				continue
			}

			count++

			fmt.Fprintf(
				&builder,
				"%s *%s*\n",
				achievement.Icon,
				achievement.Name,
			)

			description :=
				strings.TrimSpace(
					achievement.Description,
				)

			if description != "" {
				fmt.Fprintf(
					&builder,
					"%s\n\n",
					description,
				)
			}
		}

		if count == 0 {
			builder.WriteString(
				"✨ Nenhuma conquista finalizada.",
			)
		}

	case "locked":
		fmt.Fprintf(
			&builder,
			"🔒 *BLOQUEADAS DE @%s*\n\n",
			targetName,
		)

		count := 0

		for _, achievement := range achievements {

			if achievement.Unlocked ||
				achievement.Progress > 0 {

				continue
			}

			count++

			fmt.Fprintf(
				&builder,
				"🔒 %s *%s*\n",
				achievement.Icon,
				achievement.Name,
			)
		}

		if count == 0 {
			builder.WriteString(
				"✨ Nenhuma conquista bloqueada.",
			)
		}

	default:
		fmt.Fprintf(
			&builder,
			"🏆 *CONQUISTAS ATIVAS DE @%s*\n\n",
			targetName,
		)

		count := 0

		for _, achievement := range achievements {

			if achievement.Unlocked ||
				achievement.Progress <= 0 {

				continue
			}

			progress :=
				achievement.Progress

			if progress >
				achievement.Target {

				progress =
					achievement.Target
			}

			count++

			fmt.Fprintf(
				&builder,
				"%s *%s* • %d/%d\n",
				achievement.Icon,
				achievement.Name,
				progress,
				achievement.Target,
			)

			description :=
				strings.TrimSpace(
					achievement.Description,
				)

			if description != "" {
				fmt.Fprintf(
					&builder,
					"%s\n\n",
					description,
				)
			}
		}

		if count == 0 {
			builder.WriteString(
				"✨ Nenhuma conquista em andamento.",
			)
		}
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
