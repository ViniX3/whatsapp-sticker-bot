package gold

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"whatsapp-sticker-bot/internal/database"
)

const (
	CoinflipMinBet = 100

	CoinflipHeads = "cara"
	CoinflipTails = "coroa"
)

var (
	ErrCoinflipInvalidBet = errors.New(
		"aposta do cara ou coroa abaixo do valor mínimo",
	)

	ErrCoinflipInvalidChoice = errors.New(
		"escolha inválida no cara ou coroa",
	)
)

type CoinflipResult struct {
	Choice  string
	Outcome string
	Won     bool

	BetAmount int
	Prize     int
	NetResult int
	Balance   int
}

func PlayCoinflip(
	groupJID string,
	jid string,
	amount int,
	choice string,
) (*CoinflipResult, error) {
	if amount < CoinflipMinBet {
		return nil, ErrCoinflipInvalidBet
	}

	choice =
		strings.ToLower(
			strings.TrimSpace(choice),
		)

	if choice != CoinflipHeads &&
		choice != CoinflipTails {

		return nil, ErrCoinflipInvalidChoice
	}

	outcome, err :=
		randomCoinflipSide()

	if err != nil {
		return nil, err
	}

	won :=
		choice == outcome

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao iniciar transação do cara ou coroa: %w",
			err,
		)
	}

	defer tx.Rollback()

	balance, err :=
		coinflipWalletBalance(
			tx,
			groupJID,
			jid,
		)

	if err == sql.ErrNoRows {
		return nil, ErrWalletNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar carteira no cara ou coroa: %w",
			err,
		)
	}

	if balance < amount {
		return nil, ErrInsufficientGold
	}

	prize := 0

	if won {
		// Retorno de 1,98x.
		//
		// Exemplo:
		// aposta 1000
		// retorno 1980
		// lucro líquido 980
		prize =
			amount * 198 / 100
	}

	netResult :=
		prize - amount

	result, err :=
		tx.Exec(`
			UPDATE group_wallets
			SET
				gold = gold + ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
			  AND gold >= ?
		`,
			netResult,
			groupJID,
			jid,
			amount,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao atualizar saldo do cara ou coroa: %w",
			err,
		)
	}

	rows, err :=
		result.RowsAffected()

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao validar saldo do cara ou coroa: %w",
			err,
		)
	}

	if rows != 1 {
		return nil, ErrInsufficientGold
	}

	transactionType :=
		"COINFLIP_LOSS"

	if won {
		transactionType =
			"COINFLIP_WIN"
	}

	description :=
		fmt.Sprintf(
			"Cara ou Coroa - escolheu %s - resultado %s",
			strings.ToUpper(choice),
			strings.ToUpper(outcome),
		)

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
			netResult,
			transactionType,
			description,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao registrar transação do cara ou coroa: %w",
			err,
		)
	}

	newBalance, err :=
		coinflipWalletBalance(
			tx,
			groupJID,
			jid,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar saldo final do cara ou coroa: %w",
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"erro ao confirmar cara ou coroa: %w",
			err,
		)
	}

	return &CoinflipResult{
		Choice:  choice,
		Outcome: outcome,
		Won:     won,

		BetAmount: amount,
		Prize:     prize,
		NetResult: netResult,
		Balance:   newBalance,
	}, nil
}

func randomCoinflipSide() (
	string,
	error,
) {
	draw, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(2),
		)

	if err != nil {
		return "",
			fmt.Errorf(
				"erro ao lançar moeda: %w",
				err,
			)
	}

	if draw.Int64() == 0 {
		return CoinflipHeads, nil
	}

	return CoinflipTails, nil
}

type coinflipQueryer interface {
	QueryRow(
		query string,
		args ...any,
	) *sql.Row
}

func coinflipWalletBalance(
	queryer coinflipQueryer,
	groupJID string,
	jid string,
) (
	int,
	error,
) {
	var balance int

	err :=
		queryer.QueryRow(`
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

	return balance, err
}
