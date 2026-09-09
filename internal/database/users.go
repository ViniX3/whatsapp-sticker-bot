package database

import (
	"database/sql"
	"fmt"
	"time"
)

type User struct {
	JID             string
	Name            string
	Gold            int
	GoldInitialized bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func GetUser(jid string) (*User, error) {
	row := DB.QueryRow(`
		SELECT
			jid,
			name,
			gold,
			gold_initialized,
			created_at,
			updated_at
		FROM users
		WHERE jid = ?
	`, jid)

	var u User
	var initialized int

	err := row.Scan(
		&u.JID,
		&u.Name,
		&u.Gold,
		&initialized,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}

	u.GoldInitialized = initialized == 1

	return &u, nil
}

func CreateUser(jid, name string) error {
	_, err := DB.Exec(`
		INSERT INTO users (
			jid,
			name,
			gold,
			gold_initialized
		)
		VALUES (?, ?, 0, 0)
	`, jid, name)

	if err != nil {
		return fmt.Errorf("erro ao criar usuário: %w", err)
	}

	return nil
}

func UpdateGold(jid string, gold int) error {
	_, err := DB.Exec(`
		UPDATE users
		SET
			gold = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE jid = ?
	`, gold, jid)

	if err != nil {
		return fmt.Errorf("erro ao atualizar Gold: %w", err)
	}

	return nil
}

func SetGoldInitialized(jid string) error {
	_, err := DB.Exec(`
		UPDATE users
		SET
			gold_initialized = 1,
			updated_at = CURRENT_TIMESTAMP
		WHERE jid = ?
	`, jid)

	if err != nil {
		return fmt.Errorf("erro ao marcar Gold inicial como recebido: %w", err)
	}

	return nil
}

func AddGold(jid string, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("quantidade de Gold inválida: %d", amount)
	}

	result, err := DB.Exec(`
		UPDATE users
		SET
			gold = gold + ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE jid = ?
	`, amount, jid)

	if err != nil {
		return fmt.Errorf("erro ao adicionar Gold: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar atualização do Gold: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("usuário não encontrado: %s", jid)
	}

	return nil
}

func RemoveGold(jid string, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("quantidade de Gold inválida: %d", amount)
	}

	result, err := DB.Exec(`
		UPDATE users
		SET
			gold = gold - ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE jid = ?
		  AND gold >= ?
	`, amount, jid, amount)

	if err != nil {
		return fmt.Errorf("erro ao remover Gold: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar remoção do Gold: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("saldo insuficiente ou usuário não encontrado")
	}

	return nil
}

func RecordGoldTransaction(
	jid string,
	amount int,
	transactionType string,
	description string,
) error {
	_, err := DB.Exec(`
		INSERT INTO gold_transactions (
			jid,
			amount,
			type,
			description
		)
		VALUES (?, ?, ?, ?)
	`, jid, amount, transactionType, description)

	if err != nil {
		return fmt.Errorf("erro ao registrar transação Gold: %w", err)
	}

	return nil
}
