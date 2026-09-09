package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init() error {
	var err error

	DB, err = sql.Open("sqlite3", "storage/gold.db")
	if err != nil {
		return fmt.Errorf("erro ao abrir banco: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("erro ao conectar ao banco: %w", err)
	}

	return createTables()
}

func createTables() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			jid TEXT PRIMARY KEY,
			name TEXT,
			gold INTEGER NOT NULL DEFAULT 0,
			gold_initialized INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela users: %w", err)
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS gold_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			jid TEXT NOT NULL,
			amount INTEGER NOT NULL,
			type TEXT NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela gold_transactions: %w", err)
	}

	_, err = DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_gold_transactions_jid
		ON gold_transactions(jid);
	`)
	if err != nil {
		return fmt.Errorf("erro ao criar índice de transações: %w", err)
	}

	return nil
}
