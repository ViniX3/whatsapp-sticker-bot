package gold

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

const InitialGold = 1000

const (
	RobSuccessChance  = 60
	RobStealPercent   = 20
	RobFailurePenalty = 10
	RobCooldown       = 60 * time.Second
)

var (
	ErrInsufficientGold = errors.New("saldo insuficiente")
	ErrWalletNotFound   = errors.New("usuário não possui carteira Gold")
	ErrRobCooldown      = errors.New("roubo em cooldown")
)

// BetResult representa o resultado de uma aposta.
type BetResult struct {
	Result     string
	Multiplier int
	BetAmount  int
	Prize      int
	NetResult  int
	Balance    int
}

// RobResult representa o resultado de uma tentativa de roubo.
type RobResult struct {
	Success       bool
	Amount        int
	Penalty       int
	RobberBalance int
	TargetBalance int
}

// Controle de cooldown do !roubar.
var robCooldowns = struct {
	sync.Mutex
	lastAttempt map[string]time.Time
}{
	lastAttempt: make(map[string]time.Time),
}

// GetOrCreateWallet obtém a carteira existente ou cria uma nova.
//
// IMPORTANTE:
// Criar uma carteira NÃO concede Gold.
// O Gold inicial só é concedido através de ClaimInitialGold.
func GetOrCreateWallet(jid, name string) (*database.User, bool, error) {
	user, err := database.GetUser(jid)
	if err != nil {
		return nil, false, err
	}

	if user != nil {
		return user, false, nil
	}

	if err := database.CreateUser(jid, name); err != nil {
		return nil, false, err
	}

	user, err = database.GetUser(jid)
	if err != nil {
		return nil, false, err
	}

	return user, true, nil
}

// ClaimInitialGold concede os 1000 Gold iniciais apenas uma vez.
//
// Retorna:
//   - user: carteira atualizada
//   - claimed: true se os 1000 Gold foram concedidos nesta chamada
func ClaimInitialGold(jid, name string) (*database.User, bool, error) {
	tx, err := database.DB.Begin()
	if err != nil {
		return nil, false, fmt.Errorf("erro ao iniciar transação Gold: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var gold int
	var initialized int

	queryErr := tx.QueryRow(`
		SELECT gold, gold_initialized
		FROM users
		WHERE jid = ?
	`, jid).Scan(&gold, &initialized)

	if queryErr == sql.ErrNoRows {
		_, err = tx.Exec(`
			INSERT INTO users (
				jid,
				name,
				gold,
				gold_initialized
			)
			VALUES (?, ?, ?, 1)
		`, jid, name, InitialGold)

		if err != nil {
			return nil, false, fmt.Errorf(
				"erro ao criar carteira e conceder Gold inicial: %w",
				err,
			)
		}

		_, err = tx.Exec(`
			INSERT INTO gold_transactions (
				jid,
				amount,
				type,
				description
			)
			VALUES (?, ?, ?, ?)
		`,
			jid,
			InitialGold,
			"INITIAL_GOLD",
			"Gold inicial da carteira",
		)

		if err != nil {
			return nil, false, fmt.Errorf(
				"erro ao registrar Gold inicial: %w",
				err,
			)
		}

		if err = tx.Commit(); err != nil {
			return nil, false, fmt.Errorf(
				"erro ao confirmar Gold inicial: %w",
				err,
			)
		}

		user, err := database.GetUser(jid)
		if err != nil {
			return nil, false, err
		}

		return user, true, nil
	}

	if queryErr != nil {
		return nil, false, fmt.Errorf(
			"erro ao verificar carteira: %w",
			queryErr,
		)
	}

	if initialized == 1 {
		if err = tx.Commit(); err != nil {
			return nil, false, fmt.Errorf(
				"erro ao finalizar consulta da carteira: %w",
				err,
			)
		}

		user, err := database.GetUser(jid)
		if err != nil {
			return nil, false, err
		}

		return user, false, nil
	}

	result, err := tx.Exec(`
		UPDATE users
		SET
			gold = gold + ?,
			gold_initialized = 1,
			updated_at = CURRENT_TIMESTAMP
		WHERE jid = ?
		  AND gold_initialized = 0
	`, InitialGold, jid)

	if err != nil {
		return nil, false, fmt.Errorf(
			"erro ao conceder Gold inicial: %w",
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, false, fmt.Errorf(
			"erro ao verificar concessão do Gold: %w",
			err,
		)
	}

	if rows == 0 {
		if err = tx.Commit(); err != nil {
			return nil, false, err
		}

		user, err := database.GetUser(jid)
		if err != nil {
			return nil, false, err
		}

		return user, false, nil
	}

	_, err = tx.Exec(`
		INSERT INTO gold_transactions (
			jid,
			amount,
			type,
			description
		)
		VALUES (?, ?, ?, ?)
	`,
		jid,
		InitialGold,
		"INITIAL_GOLD",
		"Gold inicial da carteira",
	)

	if err != nil {
		return nil, false, fmt.Errorf(
			"erro ao registrar Gold inicial: %w",
			err,
		)
	}

	if err = tx.Commit(); err != nil {
		return nil, false, fmt.Errorf(
			"erro ao confirmar concessão do Gold: %w",
			err,
		)
	}

	user, err := database.GetUser(jid)
	if err != nil {
		return nil, false, err
	}

	return user, true, nil
}

// GetBalance retorna o saldo atual do usuário.
func GetBalance(jid string) (int, error) {
	user, err := database.GetUser(jid)
	if err != nil {
		return 0, err
	}

	if user == nil {
		return 0, nil
	}

	return user.Gold, nil
}

// Bet realiza uma aposta utilizando o sistema de multiplicadores.
//
// Probabilidades:
//
//	💀 Perdeu           55%
//	😐 Recuperou        25%
//	🍀 Pequeno prêmio   14%
//	💰 Grande prêmio     5%
//	🔥 Jackpot           0,9%
//	👑 Mega Jackpot      0,1%
func Bet(jid string, amount int) (*BetResult, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("valor da aposta deve ser maior que zero")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return nil, fmt.Errorf("erro ao realizar sorteio: %w", err)
	}

	draw := n.Int64()

	var result string
	var multiplier int

	switch {
	case draw < 5500:
		result = "💀 PERDEU"
		multiplier = 0

	case draw < 8000:
		result = "😐 RECUPEROU"
		multiplier = 1

	case draw < 9400:
		result = "🍀 PEQUENO PRÊMIO"
		multiplier = 2

	case draw < 9900:
		result = "💰 GRANDE PRÊMIO"
		multiplier = 5

	case draw < 9990:
		result = "🔥 JACKPOT"
		multiplier = 10

	default:
		result = "👑 MEGA JACKPOT"
		multiplier = 50
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao iniciar transação da aposta: %w",
			err,
		)
	}

	defer tx.Rollback()

	var balance int

	err = tx.QueryRow(`
		SELECT gold
		FROM users
		WHERE jid = ?
	`, jid).Scan(&balance)

	if err == sql.ErrNoRows {
		return nil, ErrWalletNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar saldo: %w",
			err,
		)
	}

	if balance < amount {
		return nil, ErrInsufficientGold
	}

	prize := amount * multiplier
	netResult := prize - amount
	newBalance := balance + netResult

	_, err = tx.Exec(`
		UPDATE users
		SET
			gold = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE jid = ?
	`, newBalance, jid)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao atualizar saldo da aposta: %w",
			err,
		)
	}

	transactionType := "BET_LOSS"

	if netResult >= 0 {
		transactionType = "BET_WIN"
	}

	description := fmt.Sprintf(
		"Aposta de %d Gold - %s - multiplicador %dx",
		amount,
		result,
		multiplier,
	)

	_, err = tx.Exec(`
		INSERT INTO gold_transactions (
			jid,
			amount,
			type,
			description
		)
		VALUES (?, ?, ?, ?)
	`,
		jid,
		netResult,
		transactionType,
		description,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao registrar aposta: %w",
			err,
		)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"erro ao confirmar aposta: %w",
			err,
		)
	}

	return &BetResult{
		Result:     result,
		Multiplier: multiplier,
		BetAmount:  amount,
		Prize:      prize,
		NetResult:  netResult,
		Balance:    newBalance,
	}, nil
}

// CanRob verifica se o usuário pode realizar uma tentativa de roubo.
func CanRob(jid string) bool {
	robCooldowns.Lock()
	defer robCooldowns.Unlock()

	lastAttempt, exists := robCooldowns.lastAttempt[jid]

	if !exists {
		return true
	}

	return time.Since(lastAttempt) >= RobCooldown
}

// GetRobCooldown retorna o tempo restante do cooldown.
func GetRobCooldown(jid string) time.Duration {
	robCooldowns.Lock()
	defer robCooldowns.Unlock()

	lastAttempt, exists := robCooldowns.lastAttempt[jid]

	if !exists {
		return 0
	}

	remaining := RobCooldown - time.Since(lastAttempt)

	if remaining < 0 {
		return 0
	}

	return remaining
}

// Rob realiza uma tentativa de roubo.
//
// Sucesso:
//   - 60% de chance
//   - rouba 20% do saldo da vítima
//
// Falha:
//   - paga 10% do próprio saldo
//
// A operação financeira é realizada dentro de uma única
// transação SQLite.
func Rob(robberJID, targetJID string) (*RobResult, error) {

	if robberJID == targetJID {
		return nil, fmt.Errorf("não é possível roubar a si mesmo")
	}

	if !CanRob(robberJID) {
		return nil, ErrRobCooldown
	}

	// Sorteio de 0 a 99.
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		return nil, fmt.Errorf("erro ao realizar sorteio do roubo: %w", err)
	}

	success := n.Int64() < RobSuccessChance

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao iniciar transação do roubo: %w",
			err,
		)
	}

	defer tx.Rollback()

	var robberBalance int
	var targetBalance int

	err = tx.QueryRow(`
		SELECT gold
		FROM users
		WHERE jid = ?
	`, robberJID).Scan(&robberBalance)

	if err == sql.ErrNoRows {
		return nil, ErrWalletNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar saldo do ladrão: %w",
			err,
		)
	}

	err = tx.QueryRow(`
		SELECT gold
		FROM users
		WHERE jid = ?
	`, targetJID).Scan(&targetBalance)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("vítima não possui carteira Gold")
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar saldo da vítima: %w",
			err,
		)
	}

	if targetBalance <= 0 {
		return nil, fmt.Errorf("vítima não possui Gold para ser roubado")
	}

	// =========================
	// ROUBO BEM-SUCEDIDO
	// =========================
	if success {

		amount := targetBalance * RobStealPercent / 100

		// Garante que sempre exista um roubo mínimo de 1 Gold.
		if amount < 1 {
			amount = 1
		}

		// Segurança adicional.
		if amount > targetBalance {
			amount = targetBalance
		}

		result, err := tx.Exec(`
			UPDATE users
			SET
				gold = gold - ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE jid = ?
			  AND gold >= ?
		`, amount, targetJID, amount)

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao remover Gold da vítima: %w",
				err,
			)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf(
				"erro ao verificar remoção do Gold da vítima: %w",
				err,
			)
		}

		if rows == 0 {
			return nil, fmt.Errorf(
				"saldo da vítima mudou durante a operação",
			)
		}

		_, err = tx.Exec(`
			UPDATE users
			SET
				gold = gold + ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE jid = ?
		`, amount, robberJID)

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao adicionar Gold ao ladrão: %w",
				err,
			)
		}

		newRobberBalance := robberBalance + amount
		newTargetBalance := targetBalance - amount

		_, err = tx.Exec(`
			INSERT INTO gold_transactions (
				jid,
				amount,
				type,
				description
			)
			VALUES (?, ?, ?, ?)
		`,
			targetJID,
			-amount,
			"ROB_LOSS",
			fmt.Sprintf(
				"Perdeu %d Gold em um roubo realizado por %s",
				amount,
				robberJID,
			),
		)

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao registrar perda da vítima: %w",
				err,
			)
		}

		_, err = tx.Exec(`
			INSERT INTO gold_transactions (
				jid,
				amount,
				type,
				description
			)
			VALUES (?, ?, ?, ?)
		`,
			robberJID,
			amount,
			"ROB_SUCCESS",
			fmt.Sprintf(
				"Roubo bem-sucedido contra %s - +%d Gold",
				targetJID,
				amount,
			),
		)

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao registrar ganho do ladrão: %w",
				err,
			)
		}

		if err = tx.Commit(); err != nil {
			return nil, fmt.Errorf(
				"erro ao confirmar roubo: %w",
				err,
			)
		}

		setRobCooldown(robberJID)

		return &RobResult{
			Success:       true,
			Amount:        amount,
			Penalty:       0,
			RobberBalance: newRobberBalance,
			TargetBalance: newTargetBalance,
		}, nil
	}

	// =========================
	// ROUBO MAL-SUCEDIDO
	// =========================

	if robberBalance <= 0 {
		return nil, ErrInsufficientGold
	}

	penalty := robberBalance * RobFailurePenalty / 100

	// Garante penalidade mínima de 1 Gold.
	if penalty < 1 {
		penalty = 1
	}

	// Nunca permite saldo negativo.
	if penalty > robberBalance {
		penalty = robberBalance
	}

	result, err := tx.Exec(`
		UPDATE users
		SET
			gold = gold - ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE jid = ?
		  AND gold >= ?
	`, penalty, robberJID, penalty)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao aplicar penalidade do roubo: %w",
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao verificar penalidade do roubo: %w",
			err,
		)
	}

	if rows == 0 {
		return nil, ErrInsufficientGold
	}

	newRobberBalance := robberBalance - penalty

	_, err = tx.Exec(`
		INSERT INTO gold_transactions (
			jid,
			amount,
			type,
			description
		)
		VALUES (?, ?, ?, ?)
	`,
		robberJID,
		-penalty,
		"ROB_FAIL",
		fmt.Sprintf(
			"Roubo fracassado contra %s - -%d Gold",
			targetJID,
			penalty,
		),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao registrar penalidade do roubo: %w",
			err,
		)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"erro ao confirmar penalidade do roubo: %w",
			err,
		)
	}

	setRobCooldown(robberJID)

	return &RobResult{
		Success:       false,
		Amount:        0,
		Penalty:       penalty,
		RobberBalance: newRobberBalance,
		TargetBalance: targetBalance,
	}, nil
}

// setRobCooldown registra o momento da última tentativa válida.
func setRobCooldown(jid string) {
	robCooldowns.Lock()
	defer robCooldowns.Unlock()

	robCooldowns.lastAttempt[jid] = time.Now()
}
