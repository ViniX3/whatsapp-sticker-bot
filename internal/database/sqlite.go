package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init() error {
	var err error

	// ==========================================================
	// SQLITE
	// ==========================================================
	//
	// _busy_timeout=5000
	//   Aguarda até 5 segundos caso o banco esteja ocupado.
	//
	// _journal_mode=WAL
	//   Melhora a convivência entre leituras e escritas.
	//
	// _synchronous=NORMAL
	//   Reduz I/O mantendo boa segurança com WAL.
	//
	// _foreign_keys=ON
	//   Ativa validação de chaves estrangeiras no SQLite.
	//
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

	// O bot utiliza um único banco SQLite local.
	//
	// Mantemos uma conexão para serializar operações financeiras
	// e evitar múltiplos writers disputando locks.
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
	//
	// Guarda somente a identidade global do usuário.
	//
	// O saldo NÃO fica mais nesta tabela.
	//
	// Um mesmo usuário pode possuir carteiras independentes
	// em diversos grupos.
	//
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
	//
	// Cada combinação:
	//
	//   group_jid + jid
	//
	// representa uma carteira completamente independente.
	//
	// Exemplo:
	//
	// Grupo A + Vinícius -> 5000 Gold
	// Grupo B + Vinícius -> 3000 Gold
	//
	// gold_initialized controla se os 3000 Gold iniciais
	// daquele grupo já foram recebidos.
	//
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
	//
	// Guarda o histórico financeiro de todas as carteiras.
	//
	// group_jid
	//   identifica em qual grupo ocorreu a movimentação.
	//
	// jid
	//   usuário dono da movimentação.
	//
	// related_jid
	//   usuário relacionado à operação.
	//
	// Exemplos:
	//
	// PIX_SENT
	//   jid         = remetente
	//   related_jid = destinatário
	//
	// PIX_RECEIVED
	//   jid         = destinatário
	//   related_jid = remetente
	//
	// ROB_SUCCESS
	//   jid         = ladrão
	//   related_jid = vítima
	//
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
	// ÍNDICE - RANKING
	// ==========================================================
	//
	// Otimiza:
	//
	// SELECT ...
	// FROM group_wallets
	// WHERE group_jid = ?
	// ORDER BY gold DESC
	//
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
	// ÍNDICE - HISTÓRICO DO USUÁRIO
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
	//
	// Útil principalmente para:
	//
	// !pix
	// !roubar
	//
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

	return nil
}
