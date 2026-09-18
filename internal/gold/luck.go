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
	// Cooldown entre duas utilizações válidas do !sorte
	// no mesmo grupo.
	LuckCooldown = 24 * time.Hour
)

var (
	ErrLuckCooldown = errors.New(
		"sorte diária em cooldown",
	)
)

// ==========================================================
// RESULTADOS
// ==========================================================

// LuckResult representa o resultado de uma utilização
// bem-sucedida do comando !sorte.
type LuckResult struct {
	Tier   string
	Emoji  string
	Amount int

	Balance int

	ClaimedAt       time.Time
	NextAvailableAt time.Time
}

// LuckCooldownError contém informações sobre quanto tempo
// ainda falta para o usuário poder utilizar !sorte novamente.
//
// O Unwrap permite utilizar:
//
//	errors.Is(err, ErrLuckCooldown)
//
// e também:
//
//	errors.As(err, &cooldownErr)
type LuckCooldownError struct {
	Remaining   time.Duration
	AvailableAt time.Time
}

func (e *LuckCooldownError) Error() string {
	return fmt.Sprintf(
		"sorte diária disponível novamente em %s",
		e.Remaining,
	)
}

func (e *LuckCooldownError) Unwrap() error {
	return ErrLuckCooldown
}

// ==========================================================
// CONFIGURAÇÃO DAS RARIDADES
// ==========================================================

type luckTierConfig struct {
	Name string

	Emoji string

	MinGold int
	MaxGold int
}

// Probabilidades:
//
// ⚪ COMUM
// 55%
//
// 🟢 ESPECIAL
// 25%
//
// 🔵 RARO
// 12%
//
// 🟣 SUPER RARO
// 6%
//
// 🟡 LENDÁRIO
// 1,8%
//
// 👑 MÍTICO
// 0,2%
//
// O sorteio utiliza uma escala de 0 a 9999.
func drawLuckTier() (luckTierConfig, error) {
	n, err := rand.Int(
		rand.Reader,
		big.NewInt(10000),
	)

	if err != nil {
		return luckTierConfig{}, fmt.Errorf(
			"erro ao sortear raridade da sorte: %w",
			err,
		)
	}

	draw := n.Int64()

	switch {
	// 0 - 5499
	// 55%
	case draw < 5500:
		return luckTierConfig{
			Name:    "COMUM",
			Emoji:   "⚪",
			MinGold: 300,
			MaxGold: 700,
		}, nil

	// 5500 - 7999
	// 25%
	case draw < 8000:
		return luckTierConfig{
			Name:    "ESPECIAL",
			Emoji:   "🟢",
			MinGold: 800,
			MaxGold: 1500,
		}, nil

	// 8000 - 9199
	// 12%
	case draw < 9200:
		return luckTierConfig{
			Name:    "RARO",
			Emoji:   "🔵",
			MinGold: 1800,
			MaxGold: 3000,
		}, nil

	// 9200 - 9799
	// 6%
	case draw < 9800:
		return luckTierConfig{
			Name:    "SUPER RARO",
			Emoji:   "🟣",
			MinGold: 3500,
			MaxGold: 6000,
		}, nil

	// 9800 - 9979
	// 1,8%
	case draw < 9980:
		return luckTierConfig{
			Name:    "LENDÁRIO",
			Emoji:   "🟡",
			MinGold: 8000,
			MaxGold: 12000,
		}, nil

	// 9980 - 9999
	// 0,2%
	default:
		return luckTierConfig{
			Name:    "MÍTICO",
			Emoji:   "👑",
			MinGold: 20000,
			MaxGold: 30000,
		}, nil
	}
}

// randomGold sorteia um valor inteiro inclusivo:
//
//	min <= resultado <= max
func randomGold(
	min int,
	max int,
) (int, error) {

	if min < 0 {
		return 0, fmt.Errorf(
			"valor mínimo inválido: %d",
			min,
		)
	}

	if max < min {
		return 0, fmt.Errorf(
			"faixa de Gold inválida: %d-%d",
			min,
			max,
		)
	}

	rangeSize :=
		int64(max - min + 1)

	n, err := rand.Int(
		rand.Reader,
		big.NewInt(rangeSize),
	)

	if err != nil {
		return 0, fmt.Errorf(
			"erro ao sortear quantidade de Gold: %w",
			err,
		)
	}

	return min + int(n.Int64()), nil
}

// ==========================================================
// !SORTE
// ==========================================================

// ClaimDailyLuck executa o comando !sorte.
//
// Regras:
//
//   - exige carteira existente naquele grupo;
//   - pode ser utilizado uma vez a cada 24 horas;
//   - cooldown é independente entre grupos;
//   - sorteia uma raridade;
//   - sorteia o Gold dentro da faixa da raridade;
//   - adiciona Gold à carteira;
//   - registra a transação;
//   - grava o novo cooldown.
//
// Tudo acontece dentro de uma única transação SQLite.
func ClaimDailyLuck(
	groupJID string,
	jid string,
) (*LuckResult, error) {

	now := time.Now()

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao iniciar sorte diária: %w",
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
			"erro ao consultar carteira da sorte: %w",
			err,
		)
	}

	// ======================================================
	// COOLDOWN
	// ======================================================

	var lastClaimedAt int64

	err = tx.QueryRow(`
		SELECT last_claimed_at
		FROM daily_luck
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	).Scan(
		&lastClaimedAt,
	)

	switch {
	case err == nil:
		if lastClaimedAt > 0 {
			lastClaim :=
				time.Unix(
					lastClaimedAt,
					0,
				)

			nextAvailable :=
				lastClaim.Add(
					LuckCooldown,
				)

			if nextAvailable.After(now) {
				remaining :=
					time.Until(
						nextAvailable,
					)

				if remaining < 0 {
					remaining = 0
				}

				return nil, &LuckCooldownError{
					Remaining:   remaining,
					AvailableAt: nextAvailable,
				}
			}
		}

	case err == sql.ErrNoRows:
		// Primeira utilização.
		// Pode continuar.

	default:
		return nil, fmt.Errorf(
			"erro ao consultar cooldown da sorte: %w",
			err,
		)
	}

	// ======================================================
	// RARIDADE
	// ======================================================

	tier, err :=
		drawLuckTier()

	if err != nil {
		return nil, err
	}

	// ======================================================
	// PRÊMIO
	// ======================================================

	amount, err :=
		randomGold(
			tier.MinGold,
			tier.MaxGold,
		)

	if err != nil {
		return nil, err
	}

	newBalance :=
		balance + amount

	// ======================================================
	// ADICIONAR GOLD
	// ======================================================

	result, err := tx.Exec(`
		UPDATE group_wallets
		SET
			gold = gold + ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
	`,
		amount,
		groupJID,
		jid,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao adicionar prêmio da sorte: %w",
			err,
		)
	}

	rows, err :=
		result.RowsAffected()

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao verificar prêmio da sorte: %w",
			err,
		)
	}

	if rows == 0 {
		return nil, ErrWalletNotFound
	}

	// ======================================================
	// HISTÓRICO
	// ======================================================

	if err := recordTransactionTx(
		tx,
		groupJID,
		jid,
		"",
		amount,
		"DAILY_LUCK",
		fmt.Sprintf(
			"Sorte diária %s - +%d Gold",
			tier.Name,
			amount,
		),
	); err != nil {

		return nil, err
	}

	// ======================================================
	// COOLDOWN
	// ======================================================

	_, err = tx.Exec(`
		INSERT INTO daily_luck (
			group_jid,
			jid,
			last_claimed_at,
			last_tier,
			last_amount
		)
		VALUES (?, ?, ?, ?, ?)

		ON CONFLICT(group_jid, jid) DO UPDATE SET
			last_claimed_at = excluded.last_claimed_at,
			last_tier = excluded.last_tier,
			last_amount = excluded.last_amount,
			updated_at = CURRENT_TIMESTAMP
	`,
		groupJID,
		jid,
		now.Unix(),
		tier.Name,
		amount,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao registrar cooldown da sorte: %w",
			err,
		)
	}

	// ======================================================
	// COMMIT
	// ======================================================

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"erro ao confirmar sorte diária: %w",
			err,
		)
	}

	return &LuckResult{
		Tier:   tier.Name,
		Emoji:  tier.Emoji,
		Amount: amount,

		Balance: newBalance,

		ClaimedAt: now,

		NextAvailableAt: now.Add(
			LuckCooldown,
		),
	}, nil
}
