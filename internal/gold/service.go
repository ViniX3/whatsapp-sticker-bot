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

const InitialGold = 3000

const (
	RobSuccessChance  = 60
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

	// Mantemos declarado por compatibilidade com o handler.
	//
	// A lógica nova do escudo utiliza RobResult para informar
	// se ele resistiu ou quebrou.
	ErrTargetShielded = errors.New(
		"alvo possui escudo ativo",
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
	Result     string
	Multiplier int
	BetAmount  int
	Prize      int
	NetResult  int
	Balance    int
}

type RobResult struct {
	Success bool

	Amount  int
	Penalty int

	RobberBalance int
	TargetBalance int

	// ShieldBlocked informa que havia um escudo ativo
	// e esta tentativa não chegou ao roubo financeiro.
	ShieldBlocked bool

	// ShieldBroken informa que ESTE ataque conseguiu
	// destruir o escudo.
	//
	// Mesmo assim, nenhum Gold é roubado nesta tentativa.
	ShieldBroken bool

	// Número do ataque recebido pelo escudo atual.
	ShieldAttackNumber int

	// Probabilidade percentual de quebra usada
	// nesta tentativa.
	ShieldBreakChance int
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

	switch {
	case draw < 5000:
		result = "💀 PERDEU"
		multiplier = 0

	case draw < 7500:
		result = "😐 RECUPEROU"
		multiplier = 1

	case draw < 9500:
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

	prize :=
		amount * multiplier

	netResult :=
		prize - amount

	newBalance :=
		balance + netResult

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
		BetAmount:  amount,
		Prize:      prize,
		NetResult:  netResult,
		Balance:    newBalance,
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
//  3. verifica escudo;
//  4. se houver escudo, processa desgaste/quebra;
//  5. somente sem escudo ocorre o sorteio do roubo.
//
// Se o escudo quebrar:
//
//   - nenhum Gold é roubado;
//   - o ataque é considerado concluído;
//   - o ladrão entra em cooldown;
//   - o próximo ataque poderá roubar normalmente.
//
// Usuários com saldo 0 podem tentar roubar.
func Rob(
	groupJID string,
	robberJID string,
	targetJID string,
) (*RobResult, error) {

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

	// ======================================================
	// ESCUDO
	// ======================================================
	//
	// O escudo é verificado antes da validação de saldo 0.
	//
	// Portanto, uma pessoa com escudo ativo continua
	// protegida e o ataque desgasta o escudo mesmo que
	// naquele momento esteja sem Gold.
	//
	shieldResult, err :=
		processShieldAttackTx(
			tx,
			groupJID,
			targetJID,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao processar escudo: %w",
			err,
		)
	}

	if shieldResult.Protected {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf(
				"erro ao confirmar ataque contra escudo: %w",
				err,
			)
		}

		robCompleted = true

		return &RobResult{
			Success: false,

			Amount:  0,
			Penalty: 0,

			RobberBalance: robberBalance,
			TargetBalance: targetBalance,

			ShieldBlocked: true,
			ShieldBroken:  shieldResult.Broken,

			ShieldAttackNumber: shieldResult.AttackNumber,
			ShieldBreakChance:  shieldResult.BreakChance,
		}, nil
	}

	// Não havia escudo ativo.
	//
	// Agora verificamos se existe Gold para ser roubado.
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
		n.Int64() < RobSuccessChance

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

			ShieldBlocked: false,
			ShieldBroken:  false,
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

		ShieldBlocked: false,
		ShieldBroken:  false,
	}, nil
}
