package gold

import (
	"database/sql"
	"fmt"

	"whatsapp-sticker-bot/internal/database"
)

func RewardForca(
	groupJID string,
	jid string,
	amount int,
) (int, error) {
	if amount <= 0 {
		return 0,
			fmt.Errorf(
				"recompensa do Forca inválida",
			)
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro iniciando recompensa do Forca: %w",
				err,
			)
	}

	defer tx.Rollback()

	var balance int

	err =
		tx.QueryRow(`
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
		return 0,
			ErrWalletNotFound
	}

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro consultando carteira do vencedor do Forca: %w",
				err,
			)
	}

	_, err =
		tx.Exec(`
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
		return 0,
			fmt.Errorf(
				"erro creditando recompensa do Forca: %w",
				err,
			)
	}

	_, err =
		tx.Exec(`
			INSERT INTO gold_transactions (
				group_jid,
				jid,
				related_jid,
				amount,
				type,
				description
			)
			VALUES (?, ?, NULL, ?, ?, ?)
		`,
			groupJID,
			jid,
			amount,
			"HANGMAN_WIN",
			"Vitória no jogo da Forca",
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro registrando recompensa do Forca: %w",
				err,
			)
	}

	balance += amount

	if err :=
		tx.Commit(); err != nil {

		return 0,
			fmt.Errorf(
				"erro confirmando recompensa do Forca: %w",
				err,
			)
	}

	return balance, nil
}
