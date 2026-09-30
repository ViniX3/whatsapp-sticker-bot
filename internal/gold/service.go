package gold

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"whatsapp-sticker-bot/internal/chaos"
	"whatsapp-sticker-bot/internal/database"
)

const InitialGold = 3000

const (
	RobStealPercent   = 20
	RobFailurePenalty = 10
	RobCooldown       = 60 * time.Second
)

var (
	ErrWalletNotFound = errors.New(
		"usuário não possui carteira Gold neste grupo",
	)

	ErrInsufficientGold = errors.New(
		"saldo insuficiente",
	)

	ErrTargetNoGold = errors.New(
		"alvo não possui Gold disponível para ser roubado",
	)

	ErrRobCooldown = errors.New(
		"roubo em cooldown",
	)

	ErrPixSelfTransfer = errors.New(
		"não é possível enviar Gold para si mesmo",
	)
)

// ==========================================================
// RESULTADOS
// ==========================================================

type BetResult struct {
	Result string

	// Mantido inteiro para compatibilidade com
	// perfil, XP e conquistas.
	Multiplier int

	// 150 = 1,5x; 300 = 3x etc.
	PayoutPercent int

	ChaosBonus int

	ChaosBonusPercent int

	BetAmount int
	Prize     int
	NetResult int
	Balance   int
}

type RobResult struct {
	Success bool

	Amount  int
	Penalty int

	RobberBalance int
	TargetBalance int
}

type PixResult struct {
	Amount           int
	SenderBalance    int
	RecipientBalance int
}

// ==========================================================
// COOLDOWN DO !ROUBAR
// ==========================================================

var robCooldowns = struct {
	sync.Mutex
	lastAttempt map[string]time.Time
}{
	lastAttempt: make(map[string]time.Time),
}

func cooldownKey(
	groupJID string,
	jid string,
) string {
	return groupJID + "|" + jid
}

func CanRob(
	groupJID string,
	jid string,
) bool {

	key := cooldownKey(
		groupJID,
		jid,
	)

	robCooldowns.Lock()
	defer robCooldowns.Unlock()

	lastAttempt, exists :=
		robCooldowns.lastAttempt[key]

	if !exists {
		return true
	}

	return time.Since(lastAttempt) >= RobCooldown
}

func GetRobCooldown(
	groupJID string,
	jid string,
) time.Duration {

	key := cooldownKey(
		groupJID,
		jid,
	)

	robCooldowns.Lock()
	defer robCooldowns.Unlock()

	lastAttempt, exists :=
		robCooldowns.lastAttempt[key]

	if !exists {
		return 0
	}

	remaining :=
		RobCooldown - time.Since(lastAttempt)

	if remaining < 0 {
		return 0
	}

	return remaining
}

func reserveRobCooldown(
	groupJID string,
	jid string,
) bool {

	key := cooldownKey(
		groupJID,
		jid,
	)

	robCooldowns.Lock()
	defer robCooldowns.Unlock()

	now := time.Now()

	lastAttempt, exists :=
		robCooldowns.lastAttempt[key]

	if exists &&
		now.Sub(lastAttempt) < RobCooldown {

		return false
	}

	robCooldowns.lastAttempt[key] = now

	return true
}

func clearRobCooldown(
	groupJID string,
	jid string,
) {

	key := cooldownKey(
		groupJID,
		jid,
	)

	robCooldowns.Lock()
	defer robCooldowns.Unlock()

	delete(
		robCooldowns.lastAttempt,
		key,
	)
}

// ==========================================================
// HELPERS TRANSACIONAIS
// ==========================================================

func upsertUserTx(
	tx *sql.Tx,
	jid string,
	name string,
) error {
	name =
		database.NormalizeUserDisplayName(
			jid,
			name,
		)

	_, err := tx.Exec(`
		INSERT INTO users (
			jid,
			name
		)
		VALUES (?, ?)

		ON CONFLICT(jid) DO UPDATE SET
			name = CASE
				WHEN excluded.name <> ''
				THEN excluded.name
				ELSE users.name
			END,
			updated_at = CURRENT_TIMESTAMP
	`,
		jid,
		name,
	)

	if err != nil {
		return fmt.Errorf(
			"erro ao criar/atualizar usuário: %w",
			err,
		)
	}

	return nil
}

func ensureWalletTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
) error {

	_, err := tx.Exec(`
		INSERT OR IGNORE INTO group_wallets (
			group_jid,
			jid,
			gold,
			gold_initialized
		)
		VALUES (?, ?, 0, 0)
	`,
		groupJID,
		jid,
	)

	if err != nil {
		return fmt.Errorf(
			"erro ao garantir carteira Gold: %w",
			err,
		)
	}

	return nil
}

func recordTransactionTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	relatedJID string,
	amount int,
	transactionType string,
	description string,
) error {

	var related interface{}

	if relatedJID == "" {
		related = nil
	} else {
		related = relatedJID
	}

	_, err := tx.Exec(`
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
		jid,
		related,
		amount,
		transactionType,
		description,
	)

	if err != nil {
		return fmt.Errorf(
			"erro ao registrar transação Gold: %w",
			err,
		)
	}

	return nil
}

// ==========================================================
// CARTEIRA
// ==========================================================

func GetOrCreateWallet(
	groupJID string,
	jid string,
	name string,
) (*database.Wallet, bool, error) {

	return database.EnsureWallet(
		groupJID,
		jid,
		name,
	)
}

func ClaimInitialGold(
	groupJID string,
	jid string,
	name string,
) (*database.Wallet, bool, error) {

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, false, fmt.Errorf(
			"erro ao iniciar transação Gold: %w",
			err,
		)
	}

	defer tx.Rollback()

	if err := upsertUserTx(
		tx,
		jid,
		name,
	); err != nil {

		return nil, false, err
	}

	if err := ensureWalletTx(
		tx,
		groupJID,
		jid,
	); err != nil {

		return nil, false, err
	}

	result, err := tx.Exec(`
		UPDATE group_wallets
		SET
			gold = gold + ?,
			gold_initialized = 1,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
		  AND gold_initialized = 0
	`,
		InitialGold,
		groupJID,
		jid,
	)

	if err != nil {
		return nil, false, fmt.Errorf(
			"erro ao conceder Gold inicial: %w",
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, false, fmt.Errorf(
			"erro ao verificar Gold inicial: %w",
			err,
		)
	}

	claimed := rows > 0

	if claimed {
		if err := recordTransactionTx(
			tx,
			groupJID,
			jid,
			"",
			InitialGold,
			"INITIAL_GOLD",
			"Gold inicial da carteira",
		); err != nil {

			return nil, false, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf(
			"erro ao confirmar Gold inicial: %w",
			err,
		)
	}

	wallet, err := database.GetWallet(
		groupJID,
		jid,
	)
	if err != nil {
		return nil, false, err
	}

	if wallet == nil {
		return nil, false, fmt.Errorf(
			"carteira não encontrada após operação Gold",
		)
	}

	return wallet, claimed, nil
}

func GetBalance(
	groupJID string,
	jid string,
) (int, error) {

	balance, exists, err :=
		database.GetBalance(
			groupJID,
			jid,
		)

	if err != nil {
		return 0, err
	}

	if !exists {
		return 0, nil
	}

	return balance, nil
}

func GetRanking(
	groupJID string,
	limit int,
) ([]database.RankingEntry, error) {

	return database.GetRanking(
		groupJID,
		limit,
	)
}

// ==========================================================
// !BET
// ==========================================================

func Bet(
	groupJID string,
	jid string,
	amount int,
) (*BetResult, error) {

	if amount <= 0 {
		return nil, fmt.Errorf(
			"valor da aposta deve ser maior que zero",
		)
	}

	n, err := rand.Int(
		rand.Reader,
		big.NewInt(10000),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao realizar sorteio: %w",
			err,
		)
	}

	draw := n.Int64()

	var result string
	var multiplier int
	var payoutPercent int

	// Probabilidades:
	//
	// 50,0% -> derrota
	// 35,0% -> 1,5x
	//  9,0% -> 3x
	//  3,0% -> 5x
	//  1,5% -> 10x
	//  0,8% -> 20x
	//  0,5% -> 50x
	//  0,2% -> 100x
	//
	// BET 100X

	switch {
	case draw < 5000:
		result = "💀 PERDEU"
		multiplier = 0
		payoutPercent = 0

	case draw < 8500:
		result = "🍀 VITÓRIA 1,5X"

		// Mantemos 2 internamente para que perfil,
		// XP e conquistas reconheçam como vitória.
		multiplier = 2
		payoutPercent = 150

	case draw < 9400:
		result = "💰 VITÓRIA 3X"
		multiplier = 3
		payoutPercent = 300

	case draw < 9700:
		result = "🔥 GRANDE PRÊMIO 5X"
		multiplier = 5
		payoutPercent = 500

	case draw < 9850:
		result = "⚡ SUPER PRÊMIO 10X"
		multiplier = 10
		payoutPercent = 1000

	case draw < 9930:
		result = "💎 JACKPOT 20X"
		multiplier = 20
		payoutPercent = 2000

	case draw < 9980:
		result = "👑 MEGA JACKPOT 50X"
		multiplier = 50
		payoutPercent = 5000

	default:
		result = "🌟 JACKPOT SUPREMO 100X"
		multiplier = 100
		payoutPercent = 10000
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
			"erro ao consultar saldo: %w",
			err,
		)
	}

	if balance < amount {
		return nil, ErrInsufficientGold
	}

	basePrize, err :=
		safeGoldPercent(
			amount,
			payoutPercent,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"resultado financeiro excede o limite suportado: %w",
				err,
			)
	}

	baseNetResult :=
		basePrize - amount

	prize :=
		basePrize

	chaosBonus := 0
	chaosBonusPercent := 0

	// Presságio: somente o lucro real recebe bônus.
	if baseNetResult > 0 {
		totalProfit,
			bonus,
			bonusPercent,
			active :=
			chaos.ApplyGoldReward(
				baseNetResult,
			)

		if active {
			chaosBonus =
				bonus

			chaosBonusPercent =
				bonusPercent

			if totalProfit < baseNetResult ||
				bonus < 0 {

				return nil,
					ErrGoldLimitExceeded
			}

			prize,
				err =
				safeGoldAdd(
					amount,
					totalProfit,
				)

			if err != nil {
				return nil,
					fmt.Errorf(
						"prêmio com bônus do Caos excede o limite: %w",
						err,
					)
			}
		}
	}

	netResult :=
		prize - amount

	newBalance, err :=
		safeGoldAdd(
			balance,
			netResult,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"saldo final excede o limite suportado: %w",
				err,
			)
	}

	updateResult, err := tx.Exec(`
		UPDATE group_wallets
		SET
			gold = gold + ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
		  AND gold + ? >= 0
	`,
		netResult,
		groupJID,
		jid,
		netResult,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao atualizar saldo da aposta: %w",
			err,
		)
	}

	rows, err :=
		updateResult.RowsAffected()

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao verificar aposta: %w",
			err,
		)
	}

	if rows == 0 {
		return nil, ErrInsufficientGold
	}

	// Consulta novamente o saldo dentro da mesma transação.
	// Isso evita exibir um saldo calculado a partir de uma
	// leitura antiga quando existem comandos concorrentes.
	err = tx.QueryRow(`
		SELECT gold
		FROM group_wallets
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	).Scan(
		&newBalance,
	)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando saldo após atualização: %w",
				err,
			)
	}

	transactionType := "BET_LOSS"

	if netResult >= 0 {
		transactionType = "BET_WIN"
	}

	description := fmt.Sprintf(
		"Aposta de %d Gold - %s - pagamento %d%%",
		amount,
		result,
		payoutPercent,
	)

	if err := recordTransactionTx(
		tx,
		groupJID,
		jid,
		"",
		netResult,
		transactionType,
		description,
	); err != nil {

		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"erro ao confirmar aposta: %w",
			err,
		)
	}

	return &BetResult{
		Result:     result,
		Multiplier: multiplier,

		PayoutPercent: payoutPercent,

		ChaosBonus: chaosBonus,

		ChaosBonusPercent: chaosBonusPercent,

		BetAmount: amount,
		Prize:     prize,
		NetResult: netResult,
		Balance:   newBalance,
	}, nil
}

// ==========================================================
// !PIX
// ==========================================================

func Pix(
	groupJID string,
	senderJID string,
	targetJID string,
	targetName string,
	amount int,
) (*PixResult, error) {

	if amount <= 0 {
		return nil, fmt.Errorf(
			"valor do PIX deve ser maior que zero",
		)
	}

	if senderJID == targetJID {
		return nil, ErrPixSelfTransfer
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao iniciar PIX: %w",
			err,
		)
	}

	defer tx.Rollback()

	var senderBalance int

	err = tx.QueryRow(`
		SELECT gold
		FROM group_wallets
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		senderJID,
	).Scan(
		&senderBalance,
	)

	if err == sql.ErrNoRows {
		return nil, ErrWalletNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar saldo do remetente: %w",
			err,
		)
	}

	if senderBalance < amount {
		return nil, ErrInsufficientGold
	}

	if err := upsertUserTx(
		tx,
		targetJID,
		targetName,
	); err != nil {

		return nil, err
	}

	if err := ensureWalletTx(
		tx,
		groupJID,
		targetJID,
	); err != nil {

		return nil, err
	}

	var targetBalance int

	err = tx.QueryRow(`
		SELECT gold
		FROM group_wallets
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		targetJID,
	).Scan(
		&targetBalance,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar saldo do destinatário: %w",
			err,
		)
	}

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
		senderJID,
		amount,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao remover Gold do remetente: %w",
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao verificar PIX do remetente: %w",
			err,
		)
	}

	if rows == 0 {
		return nil, ErrInsufficientGold
	}

	result, err = tx.Exec(`
		UPDATE group_wallets
		SET
			gold = gold + ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
	`,
		amount,
		groupJID,
		targetJID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao adicionar Gold ao destinatário: %w",
			err,
		)
	}

	rows, err = result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao verificar PIX do destinatário: %w",
			err,
		)
	}

	if rows == 0 {
		return nil, fmt.Errorf(
			"carteira do destinatário não encontrada",
		)
	}

	if err := recordTransactionTx(
		tx,
		groupJID,
		senderJID,
		targetJID,
		-amount,
		"PIX_SENT",
		fmt.Sprintf(
			"PIX enviado para %s - %d Gold",
			targetJID,
			amount,
		),
	); err != nil {

		return nil, err
	}

	if err := recordTransactionTx(
		tx,
		groupJID,
		targetJID,
		senderJID,
		amount,
		"PIX_RECEIVED",
		fmt.Sprintf(
			"PIX recebido de %s - +%d Gold",
			senderJID,
			amount,
		),
	); err != nil {

		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"erro ao confirmar PIX: %w",
			err,
		)
	}

	return &PixResult{
		Amount:           amount,
		SenderBalance:    senderBalance - amount,
		RecipientBalance: targetBalance + amount,
	}, nil
}

// ==========================================================
// !ROUBAR
// ==========================================================

// Rob realiza uma tentativa de roubo dentro do grupo.
//
// ORDEM:
//
//  1. valida carteira do ladrão;
//  2. valida carteira da vítima;
//  3. valida se a vítima possui Gold;
//  4. realiza o sorteio do roubo.
//
// Usuários com saldo 0 podem tentar roubar.
func Rob(
	groupJID string,
	robberJID string,
	targetJID string,
	successChance int,
) (*RobResult, error) {

	if successChance < 0 || successChance > 100 {
		return nil, fmt.Errorf(
			"chance de roubo inválida: %d",
			successChance,
		)
	}

	if robberJID == targetJID {
		return nil, fmt.Errorf(
			"não é possível roubar a si mesmo",
		)
	}

	if !reserveRobCooldown(
		groupJID,
		robberJID,
	) {
		return nil, ErrRobCooldown
	}

	robCompleted := false

	defer func() {
		if !robCompleted {
			clearRobCooldown(
				groupJID,
				robberJID,
			)
		}
	}()

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

	// ======================================================
	// LADRÃO
	// ======================================================

	err = tx.QueryRow(`
		SELECT gold
		FROM group_wallets
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		robberJID,
	).Scan(
		&robberBalance,
	)

	if err == sql.ErrNoRows {
		return nil, ErrWalletNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar saldo do ladrão: %w",
			err,
		)
	}

	// ======================================================
	// VÍTIMA
	// ======================================================

	err = tx.QueryRow(`
		SELECT gold
		FROM group_wallets
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		targetJID,
	).Scan(
		&targetBalance,
	)

	if err == sql.ErrNoRows {
		return nil, ErrTargetNoGold
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar saldo da vítima: %w",
			err,
		)
	}

	if targetBalance <= 0 {
		return nil, ErrTargetNoGold
	}

	// ======================================================
	// SORTEIO DO ROUBO
	// ======================================================

	n, err := rand.Int(
		rand.Reader,
		big.NewInt(100),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao realizar sorteio do roubo: %w",
			err,
		)
	}

	success :=
		n.Int64() < int64(successChance)

	// ======================================================
	// ROUBO BEM-SUCEDIDO
	// ======================================================

	if success {
		amount :=
			targetBalance *
				RobStealPercent /
				100

		if amount < 1 {
			amount = 1
		}

		if amount > targetBalance {
			amount = targetBalance
		}

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
			targetJID,
			amount,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao remover Gold da vítima: %w",
				err,
			)
		}

		rows, err :=
			result.RowsAffected()

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao verificar saldo da vítima: %w",
				err,
			)
		}

		if rows == 0 {
			return nil, ErrTargetNoGold
		}

		result, err = tx.Exec(`
			UPDATE group_wallets
			SET
				gold = gold + ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
		`,
			amount,
			groupJID,
			robberJID,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao adicionar Gold ao ladrão: %w",
				err,
			)
		}

		rows, err =
			result.RowsAffected()

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao verificar saldo do ladrão: %w",
				err,
			)
		}

		if rows == 0 {
			return nil, ErrWalletNotFound
		}

		newRobberBalance :=
			robberBalance + amount

		newTargetBalance :=
			targetBalance - amount

		if err := recordTransactionTx(
			tx,
			groupJID,
			targetJID,
			robberJID,
			-amount,
			"ROB_LOSS",
			fmt.Sprintf(
				"Perdeu %d Gold em roubo realizado por %s",
				amount,
				robberJID,
			),
		); err != nil {

			return nil, err
		}

		if err := recordTransactionTx(
			tx,
			groupJID,
			robberJID,
			targetJID,
			amount,
			"ROB_SUCCESS",
			fmt.Sprintf(
				"Roubo bem-sucedido contra %s - +%d Gold",
				targetJID,
				amount,
			),
		); err != nil {

			return nil, err
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf(
				"erro ao confirmar roubo: %w",
				err,
			)
		}

		robCompleted = true

		return &RobResult{
			Success:       true,
			Amount:        amount,
			Penalty:       0,
			RobberBalance: newRobberBalance,
			TargetBalance: newTargetBalance,
		}, nil
	}

	// ======================================================
	// ROUBO MAL-SUCEDIDO
	// ======================================================

	penalty :=
		robberBalance *
			RobFailurePenalty /
			100

	// Quem possui algum Gold perde no mínimo 1.
	//
	// Quem está com saldo 0 continua podendo roubar
	// e simplesmente não perde Gold caso falhe.
	if robberBalance > 0 &&
		penalty < 1 {

		penalty = 1
	}

	if penalty > robberBalance {
		penalty = robberBalance
	}

	if penalty > 0 {
		result, err := tx.Exec(`
			UPDATE group_wallets
			SET
				gold = gold - ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
			  AND gold >= ?
		`,
			penalty,
			groupJID,
			robberJID,
			penalty,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao aplicar penalidade do roubo: %w",
				err,
			)
		}

		rows, err :=
			result.RowsAffected()

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao verificar penalidade do roubo: %w",
				err,
			)
		}

		if rows == 0 {
			return nil, fmt.Errorf(
				"não foi possível aplicar penalidade do roubo",
			)
		}
	}

	newRobberBalance :=
		robberBalance - penalty

	if err := recordTransactionTx(
		tx,
		groupJID,
		robberJID,
		targetJID,
		-penalty,
		"ROB_FAIL",
		fmt.Sprintf(
			"Roubo fracassado contra %s - -%d Gold",
			targetJID,
			penalty,
		),
	); err != nil {

		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"erro ao confirmar penalidade do roubo: %w",
			err,
		)
	}

	robCompleted = true

	return &RobResult{
		Success:       false,
		Amount:        0,
		Penalty:       penalty,
		RobberBalance: newRobberBalance,
		TargetBalance: targetBalance,
	}, nil
}
