package handler

import (
	"fmt"

	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/profile"
	"whatsapp-sticker-bot/internal/quiz"
)

func recordQuizProgress(
	groupJID string,
	jid string,
	difficulty quiz.Difficulty,
	correct bool,
) string {
	result, err := profile.RecordQuizResult(
		groupJID,
		jid,
		difficulty,
		correct,
	)

	if err != nil {
		logger.Error(
			"Erro ao registrar progresso do Quiz:",
			err,
		)
		return ""
	}

	return progressionSuffix(result)
}

func recordBetProgress(
	groupJID string,
	jid string,
	multiplier int,
) string {
	result, err := profile.RecordBetResult(
		groupJID,
		jid,
		multiplier,
	)

	if err != nil {
		logger.Error(
			"Erro ao registrar progresso da aposta:",
			err,
		)
		return ""
	}

	return progressionSuffix(result)
}

func recordRobberyProgress(
	groupJID string,
	jid string,
	success bool,
) string {
	result, err := profile.RecordRobberyResult(
		groupJID,
		jid,
		success,
	)

	if err != nil {
		logger.Error(
			"Erro ao registrar progresso do roubo:",
			err,
		)
		return ""
	}

	return progressionSuffix(result)
}

func recordDailyLuckProgress(
	groupJID string,
	jid string,
) string {
	result, err := profile.RecordDailyLuck(
		groupJID,
		jid,
	)

	if err != nil {
		logger.Error(
			"Erro ao registrar progresso da sorte diária:",
			err,
		)
		return ""
	}

	return progressionSuffix(result)
}

func progressionSuffix(
	result *profile.XPResult,
) string {
	if result == nil ||
		result.Gained <= 0 {
		return ""
	}

	if result.After.Level >
		result.Before.Level {
		return fmt.Sprintf(
			"\n\n✨ *+%d XP*\n🎉 *LEVEL UP!* Nível *%d → %d*",
			result.Gained,
			result.Before.Level,
			result.After.Level,
		)
	}

	return fmt.Sprintf(
		"\n\n✨ *+%d XP*",
		result.Gained,
	)
}
