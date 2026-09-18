package gold

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

const (
	// ShieldPrice é o custo para ativar um novo escudo.
	ShieldPrice = 1000

	// ShieldDuration é o tempo máximo de duração do escudo.
	ShieldDuration = 12 * time.Hour
)

var (
	// O usuário tentou comprar um escudo enquanto ainda
	// possui outro ativo.
	ErrShieldAlreadyActive = errors.New(
		"usuário já possui escudo ativo",
	)
)

// ==========================================================
// RESULTADOS
// ==========================================================

// ShieldPurchaseResult representa o resultado da compra
// de um escudo.
type ShieldPurchaseResult struct {
	Price     int
	Balance   int
	ExpiresAt time.Time
}

// ShieldStatus representa o estado atual do escudo
// de um usuário naquele grupo.
type ShieldStatus struct {
	Active          bool
	AttacksReceived int
	ActivatedAt     time.Time
	ExpiresAt       time.Time
	Remaining       time.Duration
}

// shieldAttackResult é utilizado internamente pelo !roubar.
//
// Protected:
//
//	havia um escudo ativo no momento da tentativa.
//
// Broken:
//
//	o ataque atual conseguiu quebrar o escudo.
//
// AttackNumber:
//
//	número deste ataque contra o escudo atual.
//
// BreakChance:
//
//	chance percentual utilizada neste ataque.
//
// ExpiresAt:
//
//	horário original de expiração do escudo.
type shieldAttackResult struct {
	Protected    bool
	Broken       bool
	AttackNumber int
	BreakChance  int
	ExpiresAt    time.Time
}

// ==========================================================
// COMPRA DO ESCUDO
// ==========================================================

// BuyShield compra e ativa um escudo.
//
// Regras:
//
//   - custa 1000 Gold;
//   - dura até 12 horas;
//   - não pode comprar outro enquanto estiver ativo;
//   - se quebrar, pode comprar novamente imediatamente;
//   - se expirar, pode comprar novamente imediatamente;
//   - não existe cooldown de compra.
//
// A cobrança e a ativação acontecem dentro da MESMA
// transação SQLite.
func BuyShield(
	groupJID string,
	jid string,
) (*ShieldPurchaseResult, error) {

	now := time.Now()

	expiresAt :=
		now.Add(ShieldDuration)

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao iniciar compra do escudo: %w",
			err,
		)
	}

	defer tx.Rollback()

	// ======================================================
	// CARTEIRA
	// ======================================================

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
			"erro ao consultar saldo para compra do escudo: %w",
			err,
		)
	}

	if balance < ShieldPrice {
		return nil, ErrInsufficientGold
	}

	// ======================================================
	// ESCUDO EXISTENTE
	// ======================================================

	var active int
	var currentExpiresAt int64

	err = tx.QueryRow(`
		SELECT
			active,
			expires_at
		FROM shields
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	).Scan(
		&active,
		&currentExpiresAt,
	)

	switch {
	case err == nil:
		// Existe registro de escudo.

		if active == 1 &&
			currentExpiresAt > now.Unix() {

			return nil, ErrShieldAlreadyActive
		}

	case err == sql.ErrNoRows:
		// Nunca teve escudo.
		// Podemos continuar normalmente.

	default:
		return nil, fmt.Errorf(
			"erro ao consultar escudo existente: %w",
			err,
		)
	}

	// ======================================================
	// COBRANÇA
	// ======================================================

	result, err := tx.Exec(`
		UPDATE group_wallets
		SET
			gold = gold - ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
		  AND gold >= ?
	`,
		ShieldPrice,
		groupJID,
		jid,
		ShieldPrice,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao cobrar compra do escudo: %w",
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao verificar cobrança do escudo: %w",
			err,
		)
	}

	if rows == 0 {
		return nil, ErrInsufficientGold
	}

	// ======================================================
	// ATIVAÇÃO
	// ======================================================

	_, err = tx.Exec(`
		INSERT INTO shields (
			group_jid,
			jid,
			active,
			attacks_received,
			activated_at,
			expires_at
		)
		VALUES (?, ?, 1, 0, ?, ?)

		ON CONFLICT(group_jid, jid) DO UPDATE SET
			active = 1,
			attacks_received = 0,
			activated_at = excluded.activated_at,
			expires_at = excluded.expires_at,
			updated_at = CURRENT_TIMESTAMP
	`,
		groupJID,
		jid,
		now.Unix(),
		expiresAt.Unix(),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao ativar escudo: %w",
			err,
		)
	}

	// ======================================================
	// HISTÓRICO FINANCEIRO
	// ======================================================

	if err := recordTransactionTx(
		tx,
		groupJID,
		jid,
		"",
		-ShieldPrice,
		"SHIELD_PURCHASE",
		fmt.Sprintf(
			"Compra de escudo por %d Gold - duração máxima de 12 horas",
			ShieldPrice,
		),
	); err != nil {

		return nil, err
	}

	// ======================================================
	// COMMIT
	// ======================================================

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"erro ao confirmar compra do escudo: %w",
			err,
		)
	}

	return &ShieldPurchaseResult{
		Price:     ShieldPrice,
		Balance:   balance - ShieldPrice,
		ExpiresAt: expiresAt,
	}, nil
}

// ==========================================================
// STATUS DO ESCUDO
// ==========================================================

// GetShieldStatus consulta o escudo atual.
//
// Caso o escudo tenha expirado, ele é automaticamente
// marcado como inativo.
func GetShieldStatus(
	groupJID string,
	jid string,
) (*ShieldStatus, error) {

	var active int
	var attacksReceived int
	var activatedAt int64
	var expiresAt int64

	err := database.DB.QueryRow(`
		SELECT
			active,
			attacks_received,
			activated_at,
			expires_at
		FROM shields
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	).Scan(
		&active,
		&attacksReceived,
		&activatedAt,
		&expiresAt,
	)

	if err == sql.ErrNoRows {
		return &ShieldStatus{
			Active: false,
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar status do escudo: %w",
			err,
		)
	}

	now := time.Now()

	expiration :=
		time.Unix(expiresAt, 0)

	// Escudo expirou naturalmente.
	if active == 1 &&
		!expiration.After(now) {

		_, err := database.DB.Exec(`
			UPDATE shields
			SET
				active = 0,
				updated_at = CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao finalizar escudo expirado: %w",
				err,
			)
		}

		return &ShieldStatus{
			Active:          false,
			AttacksReceived: attacksReceived,
			ActivatedAt:     time.Unix(activatedAt, 0),
			ExpiresAt:       expiration,
			Remaining:       0,
		}, nil
	}

	if active != 1 {
		return &ShieldStatus{
			Active:          false,
			AttacksReceived: attacksReceived,
			ActivatedAt:     time.Unix(activatedAt, 0),
			ExpiresAt:       expiration,
			Remaining:       0,
		}, nil
	}

	remaining :=
		time.Until(expiration)

	if remaining < 0 {
		remaining = 0
	}

	return &ShieldStatus{
		Active:          true,
		AttacksReceived: attacksReceived,
		ActivatedAt:     time.Unix(activatedAt, 0),
		ExpiresAt:       expiration,
		Remaining:       remaining,
	}, nil
}

// ==========================================================
// ATAQUES AO ESCUDO
// ==========================================================

// shieldBreakChance retorna a chance percentual
// de quebra baseada no número de ataques sofridos.
//
// 1º ataque -> 2%
// 2º ataque -> 4%
// 3º ataque -> 7%
// 4º ataque -> 11%
// 5º ataque -> 16%
// 6º+      -> 22%
func shieldBreakChance(
	attackNumber int,
) int {

	switch attackNumber {
	case 1:
		return 2

	case 2:
		return 4

	case 3:
		return 7

	case 4:
		return 11

	case 5:
		return 16

	default:
		return 22
	}
}

// processShieldAttackTx verifica e processa um ataque
// contra o escudo da vítima.
//
// IMPORTANTE:
//
// Esta função recebe *sql.Tx porque será chamada diretamente
// de Rob(), dentro da transação já existente.
//
// Isso evita abrir uma segunda conexão enquanto o !roubar
// está segurando a única conexão SQLite disponível.
//
// Retorno:
//
// Protected=false
//
//	não existe escudo ativo.
//
// Protected=true / Broken=false
//
//	o escudo bloqueou o roubo.
//
// Protected=true / Broken=true
//
//	este ataque destruiu o escudo.
//
// Mesmo quando o escudo quebra, esse ataque NÃO rouba Gold.
// O próximo ataque poderá prosseguir normalmente.
func processShieldAttackTx(
	tx *sql.Tx,
	groupJID string,
	targetJID string,
) (*shieldAttackResult, error) {

	var active int
	var attacksReceived int
	var expiresAt int64

	err := tx.QueryRow(`
		SELECT
			active,
			attacks_received,
			expires_at
		FROM shields
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		targetJID,
	).Scan(
		&active,
		&attacksReceived,
		&expiresAt,
	)

	if err == sql.ErrNoRows {
		return &shieldAttackResult{
			Protected: false,
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar escudo durante roubo: %w",
			err,
		)
	}

	if active != 1 {
		return &shieldAttackResult{
			Protected: false,
		}, nil
	}

	now :=
		time.Now()

	expiration :=
		time.Unix(expiresAt, 0)

	// ======================================================
	// ESCUDO EXPIRADO
	// ======================================================

	if !expiration.After(now) {
		_, err := tx.Exec(`
			UPDATE shields
			SET
				active = 0,
				updated_at = CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			targetJID,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao finalizar escudo expirado durante roubo: %w",
				err,
			)
		}

		return &shieldAttackResult{
			Protected: false,
		}, nil
	}

	// ======================================================
	// NOVO ATAQUE
	// ======================================================

	attackNumber :=
		attacksReceived + 1

	breakChance :=
		shieldBreakChance(
			attackNumber,
		)

	// Sorteio em uma escala de 0 a 9999.
	//
	// Exemplo:
	//
	// 2% = 200 resultados favoráveis em 10000.
	draw, err := rand.Int(
		rand.Reader,
		big.NewInt(10000),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao sortear resistência do escudo: %w",
			err,
		)
	}

	breakThreshold :=
		int64(breakChance * 100)

	broken :=
		draw.Int64() < breakThreshold

	// ======================================================
	// ESCUDO QUEBROU
	// ======================================================

	if broken {
		_, err := tx.Exec(`
			UPDATE shields
			SET
				active = 0,
				attacks_received = ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
		`,
			attackNumber,
			groupJID,
			targetJID,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao quebrar escudo: %w",
				err,
			)
		}

		return &shieldAttackResult{
			Protected:    true,
			Broken:       true,
			AttackNumber: attackNumber,
			BreakChance:  breakChance,
			ExpiresAt:    expiration,
		}, nil
	}

	// ======================================================
	// ESCUDO RESISTIU
	// ======================================================

	_, err = tx.Exec(`
		UPDATE shields
		SET
			attacks_received = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
	`,
		attackNumber,
		groupJID,
		targetJID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao registrar ataque contra escudo: %w",
			err,
		)
	}

	return &shieldAttackResult{
		Protected:    true,
		Broken:       false,
		AttackNumber: attackNumber,
		BreakChance:  breakChance,
		ExpiresAt:    expiration,
	}, nil
}
