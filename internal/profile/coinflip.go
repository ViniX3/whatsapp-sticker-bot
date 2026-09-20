package profile

import (
	"database/sql"
	"fmt"

	"whatsapp-sticker-bot/internal/database"
)

type CoinflipStats struct {
	Played int
	Won    int
}

func initCoinflipSchema() error {
	_, err := database.DB.Exec(`
		CREATE TABLE IF NOT EXISTS coinflip_stats (
			group_jid TEXT NOT NULL,
			jid TEXT NOT NULL,

			played INTEGER NOT NULL DEFAULT 0
				CHECK (played >= 0),

			won INTEGER NOT NULL DEFAULT 0
				CHECK (won >= 0),

			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

			PRIMARY KEY (
				group_jid,
				jid
			),

			FOREIGN KEY (
				group_jid,
				jid
			)
				REFERENCES group_wallets(
					group_jid,
					jid
				)
				ON DELETE CASCADE
		);
	`)

	if err != nil {
		return fmt.Errorf(
			"erro ao criar tabela coinflip_stats: %w",
			err,
		)
	}

	return nil
}

func RecordCoinflipResult(
	groupJID string,
	jid string,
	won bool,
) error {
	if err := initCoinflipSchema(); err != nil {
		return err
	}

	winIncrement := 0

	if won {
		winIncrement = 1
	}

	_, err := database.DB.Exec(`
		INSERT INTO coinflip_stats (
			group_jid,
			jid,
			played,
			won
		)
		VALUES (?, ?, 1, ?)

		ON CONFLICT(group_jid, jid)
		DO UPDATE SET
			played = coinflip_stats.played + 1,
			won = coinflip_stats.won + excluded.won,
			updated_at = CURRENT_TIMESTAMP
	`,
		groupJID,
		jid,
		winIncrement,
	)

	if err != nil {
		return fmt.Errorf(
			"erro ao registrar estatística do cara ou coroa: %w",
			err,
		)
	}

	return nil
}

func GetCoinflipStats(
	groupJID string,
	jid string,
) (*CoinflipStats, error) {
	if err := initCoinflipSchema(); err != nil {
		return nil, err
	}

	stats := &CoinflipStats{}

	err := database.DB.QueryRow(`
		SELECT
			played,
			won
		FROM coinflip_stats
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	).Scan(
		&stats.Played,
		&stats.Won,
	)

	if err == sql.ErrNoRows {
		return stats, nil
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar estatísticas do cara ou coroa: %w",
			err,
		)
	}

	return stats, nil
}
