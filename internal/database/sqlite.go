package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init() error {
	var err error

	// Configurações importantes para o SQLite:
	//
	// _busy_timeout=5000
	//   Aguarda até 5 segundos quando o banco estiver temporariamente ocupado,
	//   em vez de retornar imediatamente "database is locked".
	//
	// _journal_mode=WAL
	//   Ativa Write-Ahead Logging, melhorando a convivência entre
	//   operações de leitura e escrita.
	//
	// _synchronous=NORMAL
	//   Mantém boa segurança dos dados com menor custo de I/O quando
	//   utilizado em conjunto com WAL.
	dsn := "file:storage/gold.db?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL"

	DB, err = sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("erro ao abrir banco: %w", err)
	}

	// O SQLite possui apenas um escritor por vez.
	//
	// Como o bot utiliza um único arquivo SQLite local,
	// limitar o pool para uma conexão evita múltiplas conexões
	// concorrendo por locks de escrita.
	DB.SetMaxOpenConns(1)
	DB.SetMaxIdleConns(1)

	// Valida imediatamente se a conexão pode ser estabelecida.
	if err := DB.Ping(); err != nil {
		_ = DB.Close()
		return fmt.Errorf("erro ao conectar ao banco: %w", err)
	}

	// Cria/verifica toda a estrutura necessária.
	if err := createTables(); err != nil {
		_ = DB.Close()
		return err
	}

	return nil
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
		return fmt.Errorf(
			"erro ao criar tabela gold_transactions: %w",
			err,
		)
	}

	_, err = DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_gold_transactions_jid
		ON gold_transactions(jid);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar índice de transações: %w",
			err,
		)
	}

	return nil
}
