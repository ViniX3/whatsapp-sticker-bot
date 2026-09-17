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
//
// O mutex protege tanto a leitura quanto a gravação do mapa.
// A reserva do cooldown é feita de maneira atômica para evitar
// duas tentativas simultâneas do mesmo usuário.
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
//
// INSERT OR IGNORE torna a operação idempotente:
// duas chamadas simultâneas para o mesmo JID não causam
// erro de PRIMARY KEY.
func GetOrCreateWallet(
	jid string,
	name string,
) (*database.User, bool, error) {

	result, err := database.DB.Exec(`
		INSERT OR IGNORE INTO users (
			jid,
			name,
			gold,
			gold_initialized
		)
		VALUES (?, ?, 0, 0)
	`, jid, name)

	if err != nil {
		return nil, false, fmt.Errorf(
			"erro ao criar carteira Gold: %w",
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, false, fmt.Errorf(
			"erro ao verificar criação da carteira: %w",
			err,
		)
	}

	created := rows > 0

	user, err := database.GetUser(jid)
	if err != nil {
		return nil, false, err
	}

	if user == nil {
		return nil, false, fmt.Errorf(
			"carteira não encontrada após criação: %s",
			jid,
		)
	}

	return user, created, nil
}

// ClaimInitialGold concede os 1000 Gold iniciais apenas uma vez.
//
// Toda a operação acontece dentro de uma única transação:
//
//   - garante que a carteira exista;
//   - concede os 1000 Gold somente se gold_initialized = 0;
//   - marca gold_initialized = 1;
//   - registra a transação;
//   - confirma tudo em um único COMMIT.
//
// Se qualquer etapa falhar, o Rollback desfaz toda a operação.
//
// Retorna:
//   - user: carteira atualizada;
//   - claimed: true se os 1000 Gold foram concedidos nesta chamada.
func ClaimInitialGold(
	jid string,
	name string,
) (*database.User, bool, error) {

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, false, fmt.Errorf(
			"erro ao iniciar transação Gold: %w",
			err,
		)
	}

	// É seguro chamar Rollback mesmo depois de Commit.
	// Nesse caso o database/sql simplesmente retorna sql.ErrTxDone,
	// que é ignorado pelo defer.
	defer tx.Rollback()

	// Garante que a carteira exista.
	//
	// INSERT OR IGNORE evita disputa de PRIMARY KEY caso duas
	// chamadas tentem criar a mesma carteira simultaneamente.
	_, err = tx.Exec(`
		INSERT OR IGNORE INTO users (
			jid,
			name,
			gold,
			gold_initialized
		)
		VALUES (?, ?, 0, 0)
	`, jid, name)

	if err != nil {
		return nil, false, fmt.Errorf(
			"erro ao garantir existência da carteira: %w",
			err,
		)
	}

	// Concede o Gold somente se ainda não tiver sido inicializado.
	//
	// A condição gold_initialized = 0 é fundamental:
	// ela garante que o bônus não possa ser recebido duas vezes.
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
			"erro ao verificar concessão do Gold inicial: %w",
			err,
		)
	}

	claimed := rows > 0

	// Só registra a transação financeira quando houve
	// efetivamente uma concessão de Gold.
	if claimed {
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
	}

	if err = tx.Commit(); err != nil {
		return nil, false, fmt.Errorf(
			"erro ao confirmar operação Gold: %w",
			err,
		)
	}

	// A transação já terminou, então podemos consultar o usuário
	// normalmente usando database.GetUser.
	user, err := database.GetUser(jid)
	if err != nil {
		return nil, false, err
	}

	if user == nil {
		return nil, false, fmt.Errorf(
			"carteira não encontrada após concessão: %s",
			jid,
		)
	}

	return user, claimed, nil
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
		return nil, fmt.Errorf(
			"valor da aposta deve ser maior que zero",
		)
	}

	// Sorteio criptograficamente seguro entre 0 e 9999.
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

	// Atualiza o saldo relativamente.
	//
	// Isso é mais seguro que gravar diretamente um saldo absoluto
	// previamente calculado.
	updateResult, err := tx.Exec(`
		UPDATE users
		SET
			gold = gold + ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE jid = ?
	`, netResult, jid)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao atualizar saldo da aposta: %w",
			err,
		)
	}

	rows, err := updateResult.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf(
			"erro ao verificar atualização da aposta: %w",
			err,
		)
	}

	if rows == 0 {
		return nil, ErrWalletNotFound
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
//
// Essa função apenas consulta o cooldown.
// A reserva atômica definitiva acontece dentro de Rob().
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

// reserveRobCooldown verifica e reserva atomicamente uma tentativa.
//
// Isso impede que duas goroutines consigam passar pelo cooldown
// simultaneamente para o mesmo usuário.
func reserveRobCooldown(jid string) bool {
	robCooldowns.Lock()
	defer robCooldowns.Unlock()

	now := time.Now()

	lastAttempt, exists := robCooldowns.lastAttempt[jid]

	if exists &&
		now.Sub(lastAttempt) < RobCooldown {
		return false
	}

	robCooldowns.lastAttempt[jid] = now

	return true
}

// clearRobCooldown remove uma reserva realizada por uma tentativa
// que não chegou a executar uma operação financeira válida.
func clearRobCooldown(jid string) {
	robCooldowns.Lock()
	defer robCooldowns.Unlock()

	delete(robCooldowns.lastAttempt, jid)
}

// Rob realiza uma tentativa de roubo.
//
// Sucesso:
//   - 60% de chance;
//   - rouba 20% do saldo da vítima.
//
// Falha:
//   - paga 10% do próprio saldo.
//
// Toda a operação financeira é realizada dentro de uma única
// transação SQLite.
func Rob(
	robberJID string,
	targetJID string,
) (*RobResult, error) {

	if robberJID == targetJID {
		return nil, fmt.Errorf(
			"não é possível roubar a si mesmo",
		)
	}

	// Reserva o cooldown atomicamente.
	//
	// Mesmo que duas mensagens !roubar sejam processadas
	// simultaneamente, apenas uma consegue continuar.
	if !reserveRobCooldown(robberJID) {
		return nil, ErrRobCooldown
	}

	// Só mantemos o cooldown se a tentativa financeira for
	// efetivamente concluída.
	robCompleted := false

	defer func() {
		if !robCompleted {
			clearRobCooldown(robberJID)
		}
	}()

	// Sorteio de 0 a 99.
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
		return nil, fmt.Errorf(
			"vítima não possui carteira Gold",
		)
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar saldo da vítima: %w",
			err,
		)
	}

	if targetBalance <= 0 {
		return nil, fmt.Errorf(
			"vítima não possui Gold para ser roubado",
		)
	}

	// ==========================================================
	// ROUBO BEM-SUCEDIDO
	// ==========================================================

	if success {
		amount := targetBalance * RobStealPercent / 100

		// Garante roubo mínimo de 1 Gold.
		if amount < 1 {
			amount = 1
		}

		// Nunca permite remover mais Gold do que a vítima possui.
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
		`,
			amount,
			targetJID,
			amount,
		)

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

		result, err = tx.Exec(`
			UPDATE users
			SET
				gold = gold + ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE jid = ?
		`,
			amount,
			robberJID,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"erro ao adicionar Gold ao ladrão: %w",
				err,
			)
		}

		rows, err = result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf(
				"erro ao verificar saldo do ladrão: %w",
				err,
			)
		}

		if rows == 0 {
			return nil, ErrWalletNotFound
		}

		newRobberBalance := robberBalance + amount
		newTargetBalance := targetBalance - amount

		// Registra a perda da vítima.
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

		// Registra o ganho do ladrão.
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

		robCompleted = true

		return &RobResult{
			Success:       true,
			Amount:        amount,
			Penalty:       0,
			RobberBalance: newRobberBalance,
			TargetBalance: newTargetBalance,
		}, nil
	}

	// ==========================================================
	// ROUBO MAL-SUCEDIDO
	// ==========================================================

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
	`,
		penalty,
		robberJID,
		penalty,
	)

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

	robCompleted = true

	return &RobResult{
		Success:       false,
		Amount:        0,
		Penalty:       penalty,
		RobberBalance: newRobberBalance,
		TargetBalance: targetBalance,
	}, nil
}
