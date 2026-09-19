package profile

import (
	"fmt"

	"whatsapp-sticker-bot/internal/database"
	"whatsapp-sticker-bot/internal/quiz"
)

const (
	XPQuizSuperEasy = 10
	XPQuizEasy      = 15
	XPQuizMedium    = 25
	XPQuizHard      = 40
	XPQuizSuperHard = 70
	XPQuizInsane    = 150

	XPBetSmallPrize = 5
	XPBetBigPrize   = 15
	XPBetJackpot    = 30
	XPBetMega       = 100

	XPRobSuccess = 15
	XPDailyLuck  = 10
)

type progressDelta struct {
	XP int

	QuizCorrect    int
	QuizWrong      int
	QuizInsaneWins int

	BetsPlayed int
	BetsWon    int

	RobberiesSuccess int
	RobberiesFailed  int

	SlotsPlayed       int
	SlotsWon          int
	SlotsBiggestPrize int

	DuelsWon  int
	DuelsLost int
}

func RecordQuizResult(
	groupJID string,
	jid string,
	difficulty quiz.Difficulty,
	correct bool,
) (*XPResult, error) {
	delta := progressDelta{}

	if correct {
		delta.QuizCorrect = 1
		delta.XP = quizXP(difficulty)

		if difficulty == quiz.DifficultyInsane {
			delta.QuizInsaneWins = 1
		}
	} else {
		delta.QuizWrong = 1
	}

	return applyProgressDelta(
		groupJID,
		jid,
		delta,
	)
}

func RecordBetResult(
	groupJID string,
	jid string,
	multiplier int,
) (*XPResult, error) {
	delta := progressDelta{
		BetsPlayed: 1,
	}

	switch multiplier {
	case 2:
		delta.BetsWon = 1
		delta.XP = XPBetSmallPrize

	case 5:
		delta.BetsWon = 1
		delta.XP = XPBetBigPrize

	case 10:
		delta.BetsWon = 1
		delta.XP = XPBetJackpot

	case 50:
		delta.BetsWon = 1
		delta.XP = XPBetMega
	}

	return applyProgressDelta(
		groupJID,
		jid,
		delta,
	)
}

func RecordRobberyResult(
	groupJID string,
	jid string,
	success bool,
) (*XPResult, error) {
	delta := progressDelta{}

	if success {
		delta.RobberiesSuccess = 1
		delta.XP = XPRobSuccess
	} else {
		delta.RobberiesFailed = 1
	}

	return applyProgressDelta(
		groupJID,
		jid,
		delta,
	)
}

func RecordDailyLuck(
	groupJID string,
	jid string,
) (*XPResult, error) {
	return applyProgressDelta(
		groupJID,
		jid,
		progressDelta{
			XP: XPDailyLuck,
		},
	)
}

func quizXP(
	difficulty quiz.Difficulty,
) int {
	switch difficulty {
	case quiz.DifficultySuperEasy:
		return XPQuizSuperEasy

	case quiz.DifficultyEasy:
		return XPQuizEasy

	case quiz.DifficultyMedium:
		return XPQuizMedium

	case quiz.DifficultyHard:
		return XPQuizHard

	case quiz.DifficultySuperHard:
		return XPQuizSuperHard

	case quiz.DifficultyInsane:
		return XPQuizInsane

	default:
		return 0
	}
}

func applyProgressDelta(
	groupJID string,
	jid string,
	delta progressDelta,
) (*XPResult, error) {
	if database.DB == nil {
		return nil, fmt.Errorf(
			"banco de dados não inicializado",
		)
	}

	if delta.XP < 0 {
		return nil, fmt.Errorf(
			"quantidade de XP não pode ser negativa",
		)
	}

	if err := Ensure(
		groupJID,
		jid,
	); err != nil {
		return nil, err
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao iniciar atualização de progresso: %w",
			err,
		)
	}

	defer tx.Rollback()

	var beforeXP int

	err = tx.QueryRow(`
		SELECT xp
		FROM player_profiles
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	).Scan(
		&beforeXP,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar XP antes da atualização: %w",
			err,
		)
	}

	_, err = tx.Exec(`
		UPDATE player_profiles
		SET
			xp = xp + ?,
			quiz_correct = quiz_correct + ?,
			quiz_wrong = quiz_wrong + ?,
			quiz_insane_wins = quiz_insane_wins + ?,
			bets_played = bets_played + ?,
			bets_won = bets_won + ?,
			robberies_success = robberies_success + ?,
			robberies_failed = robberies_failed + ?,
			slots_played = slots_played + ?,
			slots_won = slots_won + ?,
			slots_biggest_prize = CASE
				WHEN ? > slots_biggest_prize THEN ?
				ELSE slots_biggest_prize
			END,
			duels_won = duels_won + ?,
			duels_lost = duels_lost + ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
	`,
		delta.XP,
		delta.QuizCorrect,
		delta.QuizWrong,
		delta.QuizInsaneWins,
		delta.BetsPlayed,
		delta.BetsWon,
		delta.RobberiesSuccess,
		delta.RobberiesFailed,
		delta.SlotsPlayed,
		delta.SlotsWon,
		delta.SlotsBiggestPrize,
		delta.SlotsBiggestPrize,
		delta.DuelsWon,
		delta.DuelsLost,
		groupJID,
		jid,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao atualizar progresso do jogador: %w",
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"erro ao confirmar progresso do jogador: %w",
			err,
		)
	}

	afterXP :=
		beforeXP + delta.XP

	return &XPResult{
		Before: LevelFromXP(beforeXP),
		After:  LevelFromXP(afterXP),
		Gained: delta.XP,
	}, nil
}
