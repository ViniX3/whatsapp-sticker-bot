package gold

import (
	"database/sql"
	"fmt"

	"whatsapp-sticker-bot/internal/database"
)

// ==========================================================
// RESULTADO DO PRÊMIO DO QUIZ
// ==========================================================

type QuizRewardResult struct {
	Amount  int
	Balance int
}

// ==========================================================
// PAGAMENTO DO QUIZ
// ==========================================================

// RewardQuiz adiciona o prêmio de um quiz acertado.
//
// Regras:
//
//   - exige carteira existente naquele grupo;
//   - amount precisa ser maior que zero;
//   - adiciona o Gold;
//   - registra QUIZ_WIN no histórico;
//   - tudo ocorre dentro da mesma transação SQLite.
func RewardQuiz(
	groupJID string,
	jid string,
	amount int,
	difficulty string,
) (*QuizRewardResult, error) {

	if amount <= 0 {
		return nil, fmt.Errorf(
			"prêmio do quiz deve ser maior que zero",
		)
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao iniciar pagamento do quiz: %w",
			err,
		)
	}

	defer tx.Rollback()

	// ======================================================
	// CARTEIRA
	// ======================================================

	var balance int

	err = tx.QueryRow(`
		SELECT gold
		FROM group_wallets
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	).Scan(
		&balance,
	)

	if err == sql.ErrNoRows {
		return nil, ErrWalletNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar carteira do quiz: %w",
			err,
		)
	}

	// ======================================================
	// ADICIONAR GOLD
	// ======================================================

	result, err := tx.Exec(`
		UPDATE group_wallets
		SET
			gold = gold + ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
	`,
		amount,
		groupJID,
		jid,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao adicionar prêmio do quiz: %w",
			err,
		)
	}

	rows, err :=
		result.RowsAffected()

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao verificar pagamento do quiz: %w",
			err,
		)
	}

	if rows == 0 {
		return nil, ErrWalletNotFound
	}

	// ======================================================
	// HISTÓRICO
	// ======================================================

	if err := recordTransactionTx(
		tx,
		groupJID,
		jid,
		"",
		amount,
		"QUIZ_WIN",
		fmt.Sprintf(
			"Quiz %s respondido corretamente - +%d Gold",
			difficulty,
			amount,
		),
	); err != nil {

		return nil, err
	}

	// ======================================================
	// COMMIT
	// ======================================================

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"erro ao confirmar prêmio do quiz: %w",
			err,
		)
	}

	return &QuizRewardResult{
		Amount:  amount,
		Balance: balance + amount,
	}, nil
}
