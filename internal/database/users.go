package database

import (
	"database/sql"
	"fmt"
	"time"
)

// User representa a identidade global de um usuário.
//
// O saldo Gold não pertence mais ao usuário globalmente.
// Cada grupo possui sua própria carteira em group_wallets.
type User struct {
	JID       string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Wallet representa a carteira Gold de um usuário
// dentro de um grupo específico.
type Wallet struct {
	GroupJID        string
	JID             string
	Gold            int
	GoldInitialized bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// RankingEntry representa uma posição no ranking
// Gold de um grupo.
type RankingEntry struct {
	JID  string
	Name string
	Gold int
}

// ==========================================================
// USERS
// ==========================================================

// GetUser retorna a identidade global de um usuário.
//
// Retorna nil caso o usuário ainda não exista.
func GetUser(jid string) (*User, error) {
	row := DB.QueryRow(`
		SELECT
			jid,
			name,
			created_at,
			updated_at
		FROM users
		WHERE jid = ?
	`, jid)

	var user User

	err := row.Scan(
		&user.JID,
		&user.Name,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao buscar usuário: %w",
			err,
		)
	}

	return &user, nil
}

// UpsertUser cria a identidade do usuário caso ela ainda
// não exista.
//
// Caso já exista, atualiza o nome exibido pelo WhatsApp.
//
// Se o nome recebido estiver vazio, o nome existente
// é preservado.
func UpsertUser(jid, name string) error {
	_, err := DB.Exec(`
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
	`, jid, name)

	if err != nil {
		return fmt.Errorf(
			"erro ao criar/atualizar usuário: %w",
			err,
		)
	}

	return nil
}

// ==========================================================
// WALLETS
// ==========================================================

// GetWallet retorna a carteira Gold do usuário
// dentro de um determinado grupo.
//
// Retorna nil caso a carteira ainda não exista.
func GetWallet(
	groupJID string,
	jid string,
) (*Wallet, error) {

	row := DB.QueryRow(`
		SELECT
			group_jid,
			jid,
			gold,
			gold_initialized,
			created_at,
			updated_at
		FROM group_wallets
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	)

	var wallet Wallet
	var initialized int

	err := row.Scan(
		&wallet.GroupJID,
		&wallet.JID,
		&wallet.Gold,
		&initialized,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao buscar carteira Gold: %w",
			err,
		)
	}

	wallet.GoldInitialized = initialized == 1

	return &wallet, nil
}

// EnsureWallet garante que o usuário e sua carteira
// existam dentro do grupo.
//
// IMPORTANTE:
// criar a carteira NÃO concede Gold.
//
// Isso permite, por exemplo, que alguém receba !pix
// antes de executar !gold.
//
// Retorna:
//   - wallet: carteira atual;
//   - created: true caso a carteira tenha sido criada agora.
func EnsureWallet(
	groupJID string,
	jid string,
	name string,
) (*Wallet, bool, error) {

	if err := UpsertUser(jid, name); err != nil {
		return nil, false, err
	}

	result, err := DB.Exec(`
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
		return nil, false, fmt.Errorf(
			"erro ao garantir carteira Gold: %w",
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

	wallet, err := GetWallet(
		groupJID,
		jid,
	)
	if err != nil {
		return nil, false, err
	}

	if wallet == nil {
		return nil, false, fmt.Errorf(
			"carteira não encontrada após criação: grupo=%s usuário=%s",
			groupJID,
			jid,
		)
	}

	return wallet, created, nil
}

// GetBalance retorna o saldo Gold do usuário
// dentro de um grupo.
//
// exists informa se a carteira existe.
func GetBalance(
	groupJID string,
	jid string,
) (
	balance int,
	exists bool,
	err error,
) {

	wallet, err := GetWallet(
		groupJID,
		jid,
	)
	if err != nil {
		return 0, false, err
	}

	if wallet == nil {
		return 0, false, nil
	}

	return wallet.Gold, true, nil
}

// SetGoldInitialized marca que o usuário já recebeu
// o Gold inicial naquele grupo.
//
// Normalmente ClaimInitialGold fará isso dentro de sua
// própria transação.
func SetGoldInitialized(
	groupJID string,
	jid string,
) error {

	result, err := DB.Exec(`
		UPDATE group_wallets
		SET
			gold_initialized = 1,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	)

	if err != nil {
		return fmt.Errorf(
			"erro ao marcar Gold inicial como recebido: %w",
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"erro ao verificar inicialização da carteira: %w",
			err,
		)
	}

	if rows == 0 {
		return fmt.Errorf(
			"carteira não encontrada: grupo=%s usuário=%s",
			groupJID,
			jid,
		)
	}

	return nil
}

// AddGold adiciona Gold à carteira de forma relativa.
//
// Utiliza:
//
//	gold = gold + ?
//
// evitando sobrescrever alterações concorrentes.
func AddGold(
	groupJID string,
	jid string,
	amount int,
) error {

	if amount <= 0 {
		return fmt.Errorf(
			"quantidade de Gold inválida: %d",
			amount,
		)
	}

	result, err := DB.Exec(`
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
		return fmt.Errorf(
			"erro ao adicionar Gold: %w",
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"erro ao verificar adição de Gold: %w",
			err,
		)
	}

	if rows == 0 {
		return fmt.Errorf(
			"carteira não encontrada: grupo=%s usuário=%s",
			groupJID,
			jid,
		)
	}

	return nil
}

// RemoveGold remove Gold somente se o usuário possuir
// saldo suficiente.
//
// A validação e a remoção acontecem na mesma instrução SQL.
func RemoveGold(
	groupJID string,
	jid string,
	amount int,
) error {

	if amount <= 0 {
		return fmt.Errorf(
			"quantidade de Gold inválida: %d",
			amount,
		)
	}

	result, err := DB.Exec(`
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
			"erro ao remover Gold: %w",
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"erro ao verificar remoção de Gold: %w",
			err,
		)
	}

	if rows == 0 {
		return fmt.Errorf(
			"saldo insuficiente ou carteira não encontrada",
		)
	}

	return nil
}

// ==========================================================
// TRANSACTIONS
// ==========================================================

// RecordGoldTransaction registra uma movimentação financeira.
//
// relatedJID pode ser vazio.
//
// Exemplos:
//
//	!pix:
//	    jid         = remetente
//	    relatedJID  = destinatário
//
//	!roubar:
//	    jid         = ladrão/vítima
//	    relatedJID  = outro participante
func RecordGoldTransaction(
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

	_, err := DB.Exec(`
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
// RANKING
// ==========================================================

// GetRanking retorna os usuários com maior saldo
// dentro de um grupo específico.
//
// O limite máximo é 100 para evitar consultas acidentais
// excessivamente grandes.
func GetRanking(
	groupJID string,
	limit int,
) ([]RankingEntry, error) {

	if limit <= 0 {
		limit = 10
	}

	if limit > 100 {
		limit = 100
	}

	rows, err := DB.Query(`
		SELECT
			gw.jid,
			u.name,
			gw.gold
		FROM group_wallets gw

		INNER JOIN users u
			ON u.jid = gw.jid

		WHERE gw.group_jid = ?

		ORDER BY
			gw.gold DESC,
			u.name COLLATE NOCASE ASC,
			gw.created_at ASC

		LIMIT ?
	`,
		groupJID,
		limit,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar ranking Gold: %w",
			err,
		)
	}

	defer rows.Close()

	ranking := make(
		[]RankingEntry,
		0,
		limit,
	)

	for rows.Next() {
		var entry RankingEntry

		if err := rows.Scan(
			&entry.JID,
			&entry.Name,
			&entry.Gold,
		); err != nil {
			return nil, fmt.Errorf(
				"erro ao ler ranking Gold: %w",
				err,
			)
		}

		ranking = append(
			ranking,
			entry,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"erro durante leitura do ranking Gold: %w",
			err,
		)
	}

	return ranking, nil
}
