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
	// Carteira do próprio usuário não existe naquele grupo.
	ErrWalletNotFound = errors.New(
		"usuário não possui carteira Gold neste grupo",
	)

	// Saldo insuficiente para a operação solicitada.
	ErrInsufficientGold = errors.New(
		"saldo insuficiente",
	)

	// A vítima não possui carteira ou está sem Gold.
	//
	// Para quem está jogando, os dois casos representam
	// a mesma situação: não há Gold disponível para roubo.
	ErrTargetNoGold = errors.New(
		"alvo não possui Gold disponível para ser roubado",
	)

	// Será utilizado quando implementarmos !escudo.
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

// PixResult representa uma transferência de Gold.
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

// cooldownKey separa o cooldown por grupo.
//
// Assim, uma tentativa de roubo no Grupo A não bloqueia
// o mesmo usuário no Grupo B.
func cooldownKey(
	groupJID string,
	jid string,
) string {
	return groupJID + "|" + jid
}

// CanRob verifica se o usuário pode realizar uma tentativa
// de roubo naquele grupo.
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

// GetRobCooldown retorna o tempo restante do cooldown
// naquele grupo.
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

// reserveRobCooldown verifica e reserva o cooldown
// atomicamente.
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

// clearRobCooldown remove uma reserva caso a tentativa
// não tenha chegado a uma operação válida.
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

// upsertUserTx garante que a identidade global exista.
//
// Caso o nome esteja vazio, preserva o nome já existente.
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

// ensureWalletTx garante que uma carteira exista.
//
// IMPORTANTE:
//
// criar a carteira NÃO concede os 3000 Gold iniciais.
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

// recordTransactionTx registra uma movimentação
// dentro da mesma transação financeira.
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

// GetOrCreateWallet obtém ou cria uma carteira naquele grupo.
//
// Criar uma carteira não concede Gold.
//
// Isso permite, por exemplo, receber !pix antes de executar
// o comando !gold.
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

// ClaimInitialGold concede os 3000 Gold iniciais
// uma única vez POR GRUPO.
//
// Toda a operação acontece em uma única transação:
//
//   - garante identidade do usuário;
//   - garante carteira no grupo;
//   - verifica gold_initialized;
//   - concede 3000 Gold;
//   - registra histórico;
//   - COMMIT.
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

// GetBalance retorna o saldo do usuário naquele grupo.
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

// GetRanking retorna o ranking Gold daquele grupo.
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

// Bet realiza uma aposta dentro do grupo atual.
//
// Probabilidades:
//
//	💀 Perdeu           55%
//	😐 Recuperou        25%
//	🍀 Pequeno prêmio   14%
//	💰 Grande prêmio     5%
//	🔥 Jackpot           0,9%
//	👑 Mega Jackpot      0,1%
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

	// Fazemos o sorteio antes de abrir a transação para
	// manter o lock financeiro pelo menor tempo possível.
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

// Pix transfere Gold entre dois usuários dentro
// do MESMO grupo.
//
// O destinatário não precisa ter utilizado !gold antes.
//
// Nesse caso sua carteira é criada com:
//
//	gold_initialized = 0
//
// Portanto ele ainda poderá receber seus 3000 Gold
// iniciais posteriormente.
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

	// Remetente precisa possuir carteira neste grupo.
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

	// Garante a identidade global do destinatário.
	if err := upsertUserTx(
		tx,
		targetJID,
		targetName,
	); err != nil {

		return nil, err
	}

	// Garante a carteira do destinatário sem conceder
	// Gold inicial.
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

	// Remove do remetente.
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

	// Adiciona ao destinatário.
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

	// Histórico do remetente.
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

	// Histórico do destinatário.
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
// Sucesso:
//   - 60% de chance;
//   - rouba 20% do saldo da vítima.
//
// Falha:
//   - perde 10% do próprio saldo;
//   - caso esteja com 0 Gold, perde 0 Gold.
//
// IMPORTANTE:
//
// Um usuário com carteira e saldo 0 pode continuar tentando
// roubar normalmente.
//
// Retornos importantes:
//
//	ErrWalletNotFound
//	    ladrão ainda não criou carteira.
//
//	ErrTargetNoGold
//	    vítima não possui Gold disponível.
//
//	ErrTargetShielded
//	    reservado para implementação do !escudo.
//
//	ErrRobCooldown
//	    ladrão ainda está em cooldown.
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

	// Não existe mais saldo mínimo para roubar.
	//
	// Quem possui carteira pode tentar o roubo mesmo
	// estando com 0 Gold.

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

	// Carteira inexistente e carteira zerada são tratadas
	// da mesma forma para quem está jogando.
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
	// FUTURO !ESCUDO
	// ======================================================
	//
	// A verificação de escudo entrará exatamente aqui,
	// antes de qualquer movimentação financeira.
	//
	// Exemplo futuro:
	//
	// shielded, broken, err := CheckShield(...)
	//
	// if shielded {
	//     return nil, ErrTargetShielded
	// }
	//
	// Se o ataque quebrar o escudo, essa tentativa não
	// roubará Gold. O próximo ataque poderá prosseguir.

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

	// Quem possui Gold sempre perde no mínimo 1 Gold
	// em um roubo fracassado.
	//
	// Quem está com saldo zerado pode tentar roubar,
	// mas naturalmente não possui Gold para perder.
	if robberBalance > 0 &&
		penalty < 1 {

		penalty = 1
	}

	if penalty > robberBalance {
		penalty = robberBalance
	}

	// Se o usuário possui Gold, aplicamos a penalidade.
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
