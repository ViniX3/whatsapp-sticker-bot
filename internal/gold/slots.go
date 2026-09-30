package gold

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"whatsapp-sticker-bot/internal/chaos"
	"whatsapp-sticker-bot/internal/database"
)

const SlotsMinBet = 100

var ErrSlotsInvalidBet = errors.New(
	"aposta do slots abaixo do valor mínimo",
)

type SlotsResult struct {
	Symbols [9]string

	Tier string

	// Multiplier permanece inteiro para compatibilidade
	// com perfil, XP e conquistas.
	Multiplier int

	// PayoutPercent representa o pagamento real:
	//
	// 150 = 1,5x
	// 200 = 2x
	// 500 = 5x
	PayoutPercent int

	ChaosBonus int

	ChaosBonusPercent int

	WinningLine string

	BetAmount int
	Prize     int
	NetResult int
	Balance   int
}

type slotOutcome struct {
	// Multiplier é utilizado apenas para compatibilidade
	// com progressão e estatísticas.
	Multiplier int

	// PayoutPercent controla o pagamento financeiro real.
	PayoutPercent int

	Symbol string
	Tier   string
}

type slotPayline struct {
	Positions [3]int
	Name      string
}

var slotSymbols = []string{
	"🍒",
	"🍋",
	"🍇",
	"🔔",
	"⭐",
	"💰",
	"💎",
	"👑",
	"🐉",
	"7️⃣",
}

var slotPaylines = []slotPayline{
	{
		Positions: [3]int{0, 1, 2},
		Name:      "Linha superior",
	},
	{
		Positions: [3]int{3, 4, 5},
		Name:      "Linha central",
	},
	{
		Positions: [3]int{6, 7, 8},
		Name:      "Linha inferior",
	},
	{
		Positions: [3]int{0, 3, 6},
		Name:      "Coluna esquerda",
	},
	{
		Positions: [3]int{1, 4, 7},
		Name:      "Coluna central",
	},
	{
		Positions: [3]int{2, 5, 8},
		Name:      "Coluna direita",
	},
	{
		Positions: [3]int{0, 4, 8},
		Name:      "Diagonal ↘️",
	},
	{
		Positions: [3]int{2, 4, 6},
		Name:      "Diagonal ↙️",
	},
}

func PlaySlots(
	groupJID string,
	jid string,
	amount int,
) (*SlotsResult, error) {
	if amount < SlotsMinBet {
		return nil,
			ErrSlotsInvalidBet
	}

	draw, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(10000),
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro ao sortear resultado do slots: %w",
				err,
			)
	}

	outcome :=
		slotOutcomeForDraw(
			draw.Int64(),
		)

	symbols,
		winningLine,
		err :=
		slotGridForOutcome(
			outcome,
		)

	if err != nil {
		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro ao iniciar transação do slots: %w",
				err,
			)
	}

	defer tx.Rollback()

	var balance int

	err =
		tx.QueryRow(`
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
		return nil,
			ErrWalletNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro ao consultar carteira no slots: %w",
				err,
			)
	}

	if balance < amount {
		return nil,
			ErrInsufficientGold
	}

	basePrize, err :=
		safeGoldPercent(
			amount,
			outcome.PayoutPercent,
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

	updateResult, err :=
		tx.Exec(`
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
		return nil,
			fmt.Errorf(
				"erro ao atualizar saldo do slots: %w",
				err,
			)
	}

	rows, err :=
		updateResult.RowsAffected()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro ao validar atualização do slots: %w",
				err,
			)
	}

	if rows != 1 {
		return nil,
			ErrInsufficientGold
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

	transactionType :=
		"SLOTS_LOSS"

	if outcome.PayoutPercent > 100 {
		transactionType =
			"SLOTS_WIN"
	}

	description :=
		fmt.Sprintf(
			"Slots 3x3 - aposta de %d Gold - %s - pagamento %d%%",
			amount,
			outcome.Tier,
			outcome.PayoutPercent,
		)

	if winningLine != "" {
		description +=
			fmt.Sprintf(
				" - %s",
				winningLine,
			)
	}

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
		return nil,
			fmt.Errorf(
				"erro ao registrar transação do slots: %w",
				err,
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro ao confirmar resultado do slots: %w",
				err,
			)
	}

	return &SlotsResult{
		Symbols: symbols,

		Tier: outcome.Tier,

		Multiplier: outcome.Multiplier,

		PayoutPercent: outcome.PayoutPercent,

		ChaosBonus: chaosBonus,

		ChaosBonusPercent: chaosBonusPercent,

		WinningLine: winningLine,

		BetAmount: amount,
		Prize:     prize,
		NetResult: netResult,
		Balance:   newBalance,
	}, nil
}

// Probabilidades:
//
//	55,00% -> 0x
//	25,00% -> 1,5x
//	12,00% -> 2x
//	 5,00% -> 5x
//	 1,50% -> 10x
//	 0,70% -> 15x
//	 0,40% -> 20x
//	 0,20% -> 30x
//	 0,10% -> 40x
//	 0,07% -> 50x
//	 0,03% -> 100x
func slotOutcomeForDraw(
	draw int64,
) slotOutcome {
	switch {
	case draw < 5500:
		return slotOutcome{
			Multiplier:    0,
			PayoutPercent: 0,
			Tier:          "NÃO FOI DESSA VEZ",
		}

	case draw < 8000:
		return slotOutcome{
			// 2 é usado apenas para registrar esta
			// rodada como vitória na progressão.
			Multiplier:    2,
			PayoutPercent: 150,
			Symbol:        "🍒",
			Tier:          "PRÊMIO 1,5X",
		}

	case draw < 9200:
		return slotOutcome{
			Multiplier:    2,
			PayoutPercent: 200,
			Symbol:        "🍋",
			Tier:          "PRÊMIO 2X",
		}

	case draw < 9700:
		return slotOutcome{
			Multiplier:    5,
			PayoutPercent: 500,
			Symbol:        "🍇",
			Tier:          "PRÊMIO 5X",
		}

	case draw < 9850:
		return slotOutcome{
			Multiplier:    10,
			PayoutPercent: 1000,
			Symbol:        "🔔",
			Tier:          "PRÊMIO 10X",
		}

	case draw < 9920:
		return slotOutcome{
			Multiplier:    15,
			PayoutPercent: 1500,
			Symbol:        "⭐",
			Tier:          "PRÊMIO 15X",
		}

	case draw < 9960:
		return slotOutcome{
			Multiplier:    20,
			PayoutPercent: 2000,
			Symbol:        "💰",
			Tier:          "PRÊMIO 20X",
		}

	case draw < 9980:
		return slotOutcome{
			Multiplier:    30,
			PayoutPercent: 3000,
			Symbol:        "💎",
			Tier:          "PRÊMIO 30X",
		}

	case draw < 9990:
		return slotOutcome{
			Multiplier:    40,
			PayoutPercent: 4000,
			Symbol:        "👑",
			Tier:          "PRÊMIO 40X",
		}

	case draw < 9997:
		return slotOutcome{
			Multiplier:    50,
			PayoutPercent: 5000,
			Symbol:        "🐉",
			Tier:          "JACKPOT 50X",
		}

	default:
		return slotOutcome{
			Multiplier:    100,
			PayoutPercent: 10000,
			Symbol:        "7️⃣",
			Tier:          "MEGA JACKPOT 100X",
		}
	}
}

func slotGridForOutcome(
	outcome slotOutcome,
) (
	[9]string,
	string,
	error,
) {
	grid, err :=
		randomLosingSlotGrid()

	if err != nil {
		return [9]string{},
			"",
			err
	}

	if outcome.Multiplier == 0 {
		return grid,
			"",
			nil
	}

	lineIndex, err :=
		randomSlotInt(
			len(slotPaylines),
		)

	if err != nil {
		return [9]string{},
			"",
			err
	}

	line :=
		slotPaylines[lineIndex]

	for _, position := range line.Positions {

		grid[position] =
			outcome.Symbol
	}

	return grid,
		line.Name,
		nil
}

func randomLosingSlotGrid() (
	[9]string,
	error,
) {
	for attempt := 0; attempt < 100; attempt++ {

		grid, err :=
			randomSlotGrid()

		if err != nil {
			return [9]string{},
				err
		}

		if !slotGridHasWinningLine(
			grid,
		) {
			return grid, nil
		}
	}

	// Fallback determinístico sem linha premiada.
	return [9]string{
		"🍒", "🍋", "🍇",
		"🔔", "⭐", "💰",
		"💎", "👑", "🐉",
	}, nil
}

func randomSlotGrid() (
	[9]string,
	error,
) {
	var grid [9]string

	for index := range grid {

		symbol, err :=
			randomSlotSymbol()

		if err != nil {
			return [9]string{},
				err
		}

		grid[index] =
			symbol
	}

	return grid, nil
}

func randomSlotSymbol() (
	string,
	error,
) {
	index, err :=
		randomSlotInt(
			len(slotSymbols),
		)

	if err != nil {
		return "",
			err
	}

	return slotSymbols[index],
		nil
}

func randomSlotInt(
	max int,
) (int, error) {
	if max <= 0 {
		return 0,
			fmt.Errorf(
				"limite aleatório inválido: %d",
				max,
			)
	}

	value, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(
				int64(max),
			),
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro ao sortear valor do slots: %w",
				err,
			)
	}

	return int(
		value.Int64(),
	), nil
}

func slotGridHasWinningLine(
	grid [9]string,
) bool {
	for _, line := range slotPaylines {
		first := grid[line.Positions[0]]

		if first == "" {
			continue
		}

		if first == grid[line.Positions[1]] &&
			first == grid[line.Positions[2]] {

			return true
		}
	}

	return false
}
