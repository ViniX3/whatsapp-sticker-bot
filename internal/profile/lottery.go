package profile

import (
	"database/sql"
	"fmt"

	"whatsapp-sticker-bot/internal/database"
)

type LotteryStats struct {
	TicketsBought int
	Wins          int
	GoldWon       int
}

func initLotterySchema() error {
	_, err := database.DB.Exec(`
		CREATE TABLE IF NOT EXISTS lottery_stats (
			group_jid TEXT NOT NULL,
			jid TEXT NOT NULL,

			tickets_bought INTEGER NOT NULL DEFAULT 0
				CHECK (tickets_bought >= 0),

			wins INTEGER NOT NULL DEFAULT 0
				CHECK (wins >= 0),

			gold_won INTEGER NOT NULL DEFAULT 0
				CHECK (gold_won >= 0),

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
				REFERENCES group_wallets (
					group_jid,
					jid
				)
				ON DELETE CASCADE
		);
	`)

	if err != nil {
		return fmt.Errorf(
			"erro ao criar tabela lottery_stats: %w",
			err,
		)
	}

	return nil
}

func RecordLotteryPurchase(
	groupJID string,
	jid string,
	quantity int,
) error {
	if quantity <= 0 {
		return fmt.Errorf(
			"quantidade de bilhetes inválida: %d",
			quantity,
		)
	}

	if err := initLotterySchema(); err != nil {
		return err
	}

	_, err := database.DB.Exec(`
		INSERT INTO lottery_stats (
			group_jid,
			jid,
			tickets_bought,
			wins,
			gold_won
		)
		VALUES (?, ?, ?, 0, 0)

		ON CONFLICT(group_jid, jid)
		DO UPDATE SET
			tickets_bought =
				lottery_stats.tickets_bought
				+ excluded.tickets_bought,

			updated_at =
				CURRENT_TIMESTAMP
	`,
		groupJID,
		jid,
		quantity,
	)

	if err != nil {
		return fmt.Errorf(
			"erro registrando compra na estatística da loteria: %w",
			err,
		)
	}

	return nil
}

func RecordLotteryWin(
	groupJID string,
	jid string,
	prize int,
) error {
	if prize <= 0 {
		return fmt.Errorf(
			"prêmio da loteria inválido: %d",
			prize,
		)
	}

	if err := initLotterySchema(); err != nil {
		return err
	}

	_, err := database.DB.Exec(`
		INSERT INTO lottery_stats (
			group_jid,
			jid,
			tickets_bought,
			wins,
			gold_won
		)
		VALUES (?, ?, 0, 1, ?)

		ON CONFLICT(group_jid, jid)
		DO UPDATE SET
			wins =
				lottery_stats.wins + 1,

			gold_won =
				lottery_stats.gold_won
				+ excluded.gold_won,

			updated_at =
				CURRENT_TIMESTAMP
	`,
		groupJID,
		jid,
		prize,
	)

	if err != nil {
		return fmt.Errorf(
			"erro registrando vitória na loteria: %w",
			err,
		)
	}

	return nil
}

func GetLotteryStats(
	groupJID string,
	jid string,
) (*LotteryStats, error) {
	if err := initLotterySchema(); err != nil {
		return nil, err
	}

	stats :=
		&LotteryStats{}

	err := database.DB.QueryRow(`
		SELECT
			tickets_bought,
			wins,
			gold_won
		FROM lottery_stats
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	).Scan(
		&stats.TicketsBought,
		&stats.Wins,
		&stats.GoldWon,
	)

	if err == sql.ErrNoRows {
		return stats, nil
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando estatísticas da loteria: %w",
				err,
			)
	}

	return stats, nil
}
