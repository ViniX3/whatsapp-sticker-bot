package gold

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"whatsapp-sticker-bot/internal/database"
)

const SlotsMinBet = 100

var ErrSlotsInvalidBet = errors.New("aposta do slots abaixo do valor mínimo")

type SlotsResult struct {
	Symbols    [3]string
	Tier       string
	Multiplier int
	BetAmount  int
	Prize      int
	NetResult  int
	Balance    int
}

type slotOutcome struct {
	Multiplier int
	Symbol     string
	Tier       string
}

var slotSymbols = []string{
	"🍒",
	"🍋",
	"🍇",
	"🔔",
	"⭐",
	"💎",
	"👑",
	"7️⃣",
}

func PlaySlots(
	groupJID string,
	jid string,
	amount int,
) (*SlotsResult, error) {
	if amount < SlotsMinBet {
		return nil, ErrSlotsInvalidBet
	}

	draw, err := rand.Int(
		rand.Reader,
		big.NewInt(10000),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao sortear resultado do slots: %w",
			err,
		)
	}

	outcome := slotOutcomeForDraw(
		draw.Int64(),
	)

	var symbols [3]string

	if outcome.Multiplier == 0 {
		symbols, err = randomLosingSymbols()
		if err != nil {
			return nil, err
		}
	} else {
		symbols = [3]string{
			outcome.Symbol,
			outcome.Symbol,
			outcome.Symbol,
		}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao iniciar transação do slots: %w",
			err,
		)
	}

	defer tx.Rollback()

	var balance int

	err = tx.QueryRow(`
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
		return nil, ErrWalletNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar carteira no slots: %w",
			err,
		)
	}

	if balance < amount {
		return nil, ErrInsufficientGold
	}

	prize :=
		amount * outcome.Multiplier

	netResult :=
		prize - amount

	newBalance :=
		balance + netResult

	result, err := tx.Exec(`
		UPDATE group_wallets
		SET
			gold = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
		  AND gold >= ?
	`,
		newBalance,
		groupJID,
		jid,
		amount,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao atualizar saldo do slots: %w",
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao validar atualização do slots: %w",
			err,
		)
	}

	if rows == 0 {
		return nil, ErrInsufficientGold
	}

	transactionType := "SLOTS_LOSS"

	switch {
	case outcome.Multiplier == 1:
		transactionType = "SLOTS_REFUND"

	case outcome.Multiplier > 1:
		transactionType = "SLOTS_WIN"
	}

	description := fmt.Sprintf(
		"Slots - aposta de %d Gold - %s - multiplicador %dx",
		amount,
		outcome.Tier,
		outcome.Multiplier,
	)

	_, err = tx.Exec(`
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
			"erro ao registrar transação do slots: %w",
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"erro ao confirmar resultado do slots: %w",
			err,
		)
	}

	return &SlotsResult{
		Symbols:    symbols,
		Tier:       outcome.Tier,
		Multiplier: outcome.Multiplier,
		BetAmount:  amount,
		Prize:      prize,
		NetResult:  netResult,
		Balance:    newBalance,
	}, nil
}

func slotOutcomeForDraw(
	draw int64,
) slotOutcome {
	switch {
	case draw < 5000:
		return slotOutcome{
			Multiplier: 0,
			Tier:       "PERDEU",
		}

	case draw < 7500:
		return slotOutcome{
			Multiplier: 1,
			Symbol:     "🍒",
			Tier:       "RECUPEROU",
		}

	case draw < 9350:
		return slotOutcome{
			Multiplier: 2,
			Symbol:     "🍋",
			Tier:       "PRÊMIO 2X",
		}

	case draw < 9750:
		return slotOutcome{
			Multiplier: 3,
			Symbol:     "🍇",
			Tier:       "PRÊMIO 3X",
		}

	case draw < 9900:
		return slotOutcome{
			Multiplier: 5,
			Symbol:     "🔔",
			Tier:       "PRÊMIO 5X",
		}

	case draw < 9970:
		return slotOutcome{
			Multiplier: 10,
			Symbol:     "⭐",
			Tier:       "JACKPOT 10X",
		}

	case draw < 9990:
		return slotOutcome{
			Multiplier: 25,
			Symbol:     "💎",
			Tier:       "SUPER JACKPOT 25X",
		}

	case draw < 9999:
		return slotOutcome{
			Multiplier: 50,
			Symbol:     "👑",
			Tier:       "MEGA JACKPOT 50X",
		}

	default:
		return slotOutcome{
			Multiplier: 100,
			Symbol:     "7️⃣",
			Tier:       "LENDÁRIO 100X",
		}
	}
}

func randomLosingSymbols() (
	[3]string,
	error,
) {
	var result [3]string

	for i := range result {
		n, err := rand.Int(
			rand.Reader,
			big.NewInt(
				int64(len(slotSymbols)),
			),
		)
		if err != nil {
			return result, fmt.Errorf(
				"erro ao sortear símbolo do slots: %w",
				err,
			)
		}

		result[i] =
			slotSymbols[int(n.Int64())]
	}

	if result[0] == result[1] &&
		result[1] == result[2] {
		for i, symbol := range slotSymbols {
			if symbol == result[2] {
				result[2] =
					slotSymbols[(i+1)%len(slotSymbols)]
				break
			}
		}
	}

	return result, nil
}
