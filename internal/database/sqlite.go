package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init() error {
	var err error

	dsn := "file:storage/gold.db?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON"

	DB, err = sql.Open(
		"sqlite3",
		dsn,
	)
	if err != nil {
		return fmt.Errorf(
			"erro ao abrir banco: %w",
			err,
		)
	}

	DB.SetMaxOpenConns(1)
	DB.SetMaxIdleConns(1)

	if err := DB.Ping(); err != nil {
		_ = DB.Close()

		return fmt.Errorf(
			"erro ao conectar ao banco: %w",
			err,
		)
	}

	if err := createTables(); err != nil {
		_ = DB.Close()

		return err
	}

	return nil
}

func createTables() error {

	// ==========================================================
	// USERS
	// ==========================================================

	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			jid TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar tabela users: %w",
			err,
		)
	}

	// ==========================================================
	// GROUP WALLETS
	// ==========================================================

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS group_wallets (
			group_jid TEXT NOT NULL,
			jid TEXT NOT NULL,

			gold INTEGER NOT NULL DEFAULT 0
				CHECK (gold >= 0),

			gold_initialized INTEGER NOT NULL DEFAULT 0
				CHECK (gold_initialized IN (0, 1)),

			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

			PRIMARY KEY (
				group_jid,
				jid
			),

			FOREIGN KEY (jid)
				REFERENCES users(jid)
				ON DELETE CASCADE
		);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar tabela group_wallets: %w",
			err,
		)
	}

	// ==========================================================
	// GOLD TRANSACTIONS
	// ==========================================================

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS gold_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,

			group_jid TEXT NOT NULL,
			jid TEXT NOT NULL,
			related_jid TEXT,

			amount INTEGER NOT NULL,

			type TEXT NOT NULL,
			description TEXT,

			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

			FOREIGN KEY (jid)
				REFERENCES users(jid)
				ON DELETE CASCADE
		);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar tabela gold_transactions: %w",
			err,
		)
	}

	// ==========================================================
	// SHIELDS
	// ==========================================================

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS shields (
			group_jid TEXT NOT NULL,
			jid TEXT NOT NULL,

			active INTEGER NOT NULL DEFAULT 0
				CHECK (active IN (0, 1)),

			attacks_received INTEGER NOT NULL DEFAULT 0
				CHECK (attacks_received >= 0),

			activated_at INTEGER NOT NULL DEFAULT 0,
			expires_at INTEGER NOT NULL DEFAULT 0,

			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

			PRIMARY KEY (
				group_jid,
				jid
			),

			FOREIGN KEY (jid)
				REFERENCES users(jid)
				ON DELETE CASCADE
		);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar tabela shields: %w",
			err,
		)
	}

	// ==========================================================
	// DAILY LUCK
	// ==========================================================
	//
	// Controla o comando !sorte.
	//
	// Cada usuário possui um cooldown independente
	// dentro de cada grupo.
	//
	// last_claimed_at:
	//   Unix timestamp da última utilização válida.
	//
	// last_tier:
	//   raridade obtida na última tentativa.
	//
	// last_amount:
	//   quantidade de Gold recebida.
	//
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS daily_luck (
			group_jid TEXT NOT NULL,
			jid TEXT NOT NULL,

			last_claimed_at INTEGER NOT NULL DEFAULT 0,

			last_tier TEXT NOT NULL DEFAULT '',

			last_amount INTEGER NOT NULL DEFAULT 0
				CHECK (last_amount >= 0),

			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

			PRIMARY KEY (
				group_jid,
				jid
			),

			FOREIGN KEY (jid)
				REFERENCES users(jid)
				ON DELETE CASCADE
		);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar tabela daily_luck: %w",
			err,
		)
	}

	// ==========================================================
	// ÍNDICE - RANKING
	// ==========================================================

	_, err = DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_group_wallets_ranking
		ON group_wallets (
			group_jid,
			gold DESC
		);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar índice de ranking: %w",
			err,
		)
	}

	// ==========================================================
	// ÍNDICE - HISTÓRICO GOLD
	// ==========================================================

	_, err = DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_gold_transactions_group_user
		ON gold_transactions (
			group_jid,
			jid,
			created_at DESC
		);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar índice de transações do usuário: %w",
			err,
		)
	}

	// ==========================================================
	// ÍNDICE - USUÁRIO RELACIONADO
	// ==========================================================

	_, err = DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_gold_transactions_related
		ON gold_transactions (
			group_jid,
			related_jid
		);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar índice de usuário relacionado: %w",
			err,
		)
	}

	// ==========================================================
	// ÍNDICE - ESCUDOS
	// ==========================================================

	_, err = DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_shields_group_active
		ON shields (
			group_jid,
			active,
			expires_at
		);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar índice de escudos: %w",
			err,
		)
	}

	// ==========================================================
	// ÍNDICE - SORTE
	// ==========================================================

	_, err = DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_daily_luck_group_claim
		ON daily_luck (
			group_jid,
			last_claimed_at
		);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar índice de sorte diária: %w",
			err,
		)
	}

	return nil
}
