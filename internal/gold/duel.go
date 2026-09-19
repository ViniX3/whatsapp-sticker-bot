package gold

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"whatsapp-sticker-bot/internal/database"
)

const DuelMinBet = 100

var (
	ErrDuelInvalidBet = errors.New(
		"aposta de duelo inválida",
	)

	ErrDuelSameUser = errors.New(
		"não é possível duelar consigo mesmo",
	)

	ErrDuelChallengerWalletNotFound = errors.New(
		"desafiante não possui carteira Gold",
	)

	ErrDuelTargetWalletNotFound = errors.New(
		"desafiado não possui carteira Gold",
	)

	ErrDuelChallengerInsufficientGold = errors.New(
		"desafiante não possui Gold suficiente",
	)

	ErrDuelTargetInsufficientGold = errors.New(
		"desafiado não possui Gold suficiente",
	)
)

type DuelResult struct {
	WinnerJID string
	LoserJID  string

	BetAmount int
	Pot       int

	WinnerBalance int
	LoserBalance  int

	ChallengerBalance int
	TargetBalance     int
}

func ValidateDuel(
	groupJID string,
	challengerJID string,
	targetJID string,
	amount int,
) error {
	if amount < DuelMinBet {
		return ErrDuelInvalidBet
	}

	if challengerJID == targetJID {
		return ErrDuelSameUser
	}

	challengerBalance, err :=
		duelWalletBalance(
			database.DB,
			groupJID,
			challengerJID,
		)

	if err == sql.ErrNoRows {
		return ErrDuelChallengerWalletNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"erro ao consultar desafiante: %w",
			err,
		)
	}

	targetBalance, err :=
		duelWalletBalance(
			database.DB,
			groupJID,
			targetJID,
		)

	if err == sql.ErrNoRows {
		return ErrDuelTargetWalletNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"erro ao consultar desafiado: %w",
			err,
		)
	}

	if challengerBalance < amount {
		return ErrDuelChallengerInsufficientGold
	}

	if targetBalance < amount {
		return ErrDuelTargetInsufficientGold
	}

	return nil
}

func ResolveDuel(
	groupJID string,
	challengerJID string,
	targetJID string,
	amount int,
) (*DuelResult, error) {
	if amount < DuelMinBet {
		return nil, ErrDuelInvalidBet
	}

	if challengerJID == targetJID {
		return nil, ErrDuelSameUser
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao iniciar transação do duelo: %w",
			err,
		)
	}

	defer tx.Rollback()

	challengerBalance, err :=
		duelWalletBalance(
			tx,
			groupJID,
			challengerJID,
		)

	if err == sql.ErrNoRows {
		return nil,
			ErrDuelChallengerWalletNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar desafiante: %w",
			err,
		)
	}

	targetBalance, err :=
		duelWalletBalance(
			tx,
			groupJID,
			targetJID,
		)

	if err == sql.ErrNoRows {
		return nil,
			ErrDuelTargetWalletNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar desafiado: %w",
			err,
		)
	}

	if challengerBalance < amount {
		return nil,
			ErrDuelChallengerInsufficientGold
	}

	if targetBalance < amount {
		return nil,
			ErrDuelTargetInsufficientGold
	}

	draw, err := rand.Int(
		rand.Reader,
		big.NewInt(2),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao sortear vencedor do duelo: %w",
			err,
		)
	}

	winnerJID := challengerJID
	loserJID := targetJID

	if draw.Int64() == 1 {
		winnerJID = targetJID
		loserJID = challengerJID
	}

	if err :=
		deductDuelBet(
			tx,
			groupJID,
			challengerJID,
			amount,
		); err != nil {

		return nil, err
	}

	if err :=
		deductDuelBet(
			tx,
			groupJID,
			targetJID,
			amount,
		); err != nil {

		return nil, err
	}

	pot :=
		amount * 2

	_, err = tx.Exec(`
		UPDATE group_wallets
		SET
			gold = gold + ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
	`,
		pot,
		groupJID,
		winnerJID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao creditar prêmio do duelo: %w",
			err,
		)
	}

	_, err = tx.Exec(`
		INSERT INTO gold_transactions (
			group_jid,
			jid,
			related_jid,
			amount,
			type,
			description
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		groupJID,
		winnerJID,
		loserJID,
		amount,
		"DUEL_WIN",
		fmt.Sprintf(
			"Vitória em duelo de %d Gold",
			amount,
		),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao registrar vitória do duelo: %w",
			err,
		)
	}

	_, err = tx.Exec(`
		INSERT INTO gold_transactions (
			group_jid,
			jid,
			related_jid,
			amount,
			type,
			description
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		groupJID,
		loserJID,
		winnerJID,
		-amount,
		"DUEL_LOSS",
		fmt.Sprintf(
			"Derrota em duelo de %d Gold",
			amount,
		),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao registrar derrota do duelo: %w",
			err,
		)
	}

	winnerBalance, err :=
		duelWalletBalance(
			tx,
			groupJID,
			winnerJID,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar saldo do vencedor: %w",
			err,
		)
	}

	loserBalance, err :=
		duelWalletBalance(
			tx,
			groupJID,
			loserJID,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar saldo do perdedor: %w",
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"erro ao confirmar duelo: %w",
			err,
		)
	}

	result := &DuelResult{
		WinnerJID: winnerJID,
		LoserJID:  loserJID,

		BetAmount: amount,
		Pot:       pot,

		WinnerBalance: winnerBalance,
		LoserBalance:  loserBalance,
	}

	if winnerJID == challengerJID {
		result.ChallengerBalance =
			winnerBalance

		result.TargetBalance =
			loserBalance
	} else {
		result.ChallengerBalance =
			loserBalance

		result.TargetBalance =
			winnerBalance
	}

	return result, nil
}

type duelQueryer interface {
	QueryRow(
		query string,
		args ...any,
	) *sql.Row
}

func duelWalletBalance(
	queryer duelQueryer,
	groupJID string,
	jid string,
) (int, error) {
	var balance int

	err := queryer.QueryRow(`
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

func deductDuelBet(
	tx *sql.Tx,
	groupJID string,
	jid string,
	amount int,
) error {
	result, err := tx.Exec(`
		UPDATE group_wallets
		SET
			gold = gold - ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
		  AND gold >= ?
	`,
		amount,
		groupJID,
		jid,
		amount,
	)

	if err != nil {
		return fmt.Errorf(
			"erro ao debitar aposta do duelo: %w",
			err,
		)
	}

	rows, err :=
		result.RowsAffected()

	if err != nil {
		return fmt.Errorf(
			"erro ao validar débito do duelo: %w",
			err,
		)
	}

	if rows != 1 {
		return ErrInsufficientGold
	}

	return nil
}
