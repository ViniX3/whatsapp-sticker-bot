package rpg

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"whatsapp-sticker-bot/internal/database"
)

const (
	VaultTierIron350K         = "IRON_350K"
	VaultTierSteel500K        = "STEEL_500K"
	VaultTierRunic750K        = "RUNIC_750K"
	VaultTierRoyal1M          = "ROYAL_1M"
	VaultTierArcane3M         = "ARCANE_3M"
	VaultTierDraconic5M       = "DRACONIC_5M"
	VaultTierImperial10M      = "IMPERIAL_10M"
	VaultTierCelestial20M     = "CELESTIAL_20M"
	VaultTierAbyssal50M       = "ABYSSAL_50M"
	VaultTierTranscendent100M = "TRANSCENDENT_100M"

	VaultTierVoidInfinite = "VOID_INFINITE"

	DefaultVaultTier = VaultTierIron350K
)

type VaultResource string

const (
	VaultResourceGold VaultResource = "GOLD"

	VaultResourceMagicCrystals VaultResource = "MAGIC_CRYSTALS"

	VaultResourceStellarStones VaultResource = "STELLAR_STONES"
)

type VaultTier struct {
	ID string

	Name string

	Capacity int

	Infinite bool

	ShopAvailable bool

	ExclusiveDrop string
}

var vaultTiers = []VaultTier{
	{
		ID:            VaultTierIron350K,
		Name:          "Cofre de Ferro",
		Capacity:      350000,
		ShopAvailable: true,
	},
	{
		ID:            VaultTierSteel500K,
		Name:          "Cofre de Aço",
		Capacity:      500000,
		ShopAvailable: true,
	},
	{
		ID:            VaultTierRunic750K,
		Name:          "Cofre Rúnico",
		Capacity:      750000,
		ShopAvailable: true,
	},
	{
		ID:            VaultTierRoyal1M,
		Name:          "Cofre Real",
		Capacity:      1000000,
		ShopAvailable: true,
	},
	{
		ID:            VaultTierArcane3M,
		Name:          "Cofre Arcano",
		Capacity:      3000000,
		ShopAvailable: true,
	},
	{
		ID:            VaultTierDraconic5M,
		Name:          "Cofre Dracônico",
		Capacity:      5000000,
		ShopAvailable: true,
	},
	{
		ID:            VaultTierImperial10M,
		Name:          "Cofre Imperial",
		Capacity:      10000000,
		ShopAvailable: true,
	},
	{
		ID:            VaultTierCelestial20M,
		Name:          "Cofre Celestial",
		Capacity:      20000000,
		ShopAvailable: true,
	},
	{
		ID:            VaultTierAbyssal50M,
		Name:          "Cofre Abissal",
		Capacity:      50000000,
		ShopAvailable: true,
	},
	{
		ID:            VaultTierTranscendent100M,
		Name:          "Cofre Transcendente",
		Capacity:      100000000,
		ShopAvailable: true,
	},
	{
		ID:            VaultTierVoidInfinite,
		Name:          "Cofre do Vazio Sem Fim",
		Infinite:      true,
		ShopAvailable: false,
		ExclusiveDrop: "DEUS_DO_VAZIO",
	},
}

var (
	ErrVaultInvalidAmount = errors.New(
		"quantidade inválida para o cofre",
	)

	ErrVaultInvalidResource = errors.New(
		"recurso inválido para o cofre",
	)

	ErrVaultUnknownTier = errors.New(
		"tipo de cofre desconhecido",
	)

	ErrVaultTierTooSmall = errors.New(
		"novo cofre não comporta os recursos atuais",
	)
)

type VaultCapacityError struct {
	Tier VaultTier

	Resource VaultResource

	Current int

	Requested int
}

func (e *VaultCapacityError) Error() string {
	return fmt.Sprintf(
		"capacidade do cofre excedida para %s",
		e.Resource,
	)
}

type VaultInsufficientError struct {
	Resource VaultResource

	Available int

	Requested int

	Stored bool
}

func (e *VaultInsufficientError) Error() string {
	if e.Stored {
		return fmt.Sprintf(
			"saldo insuficiente no cofre para %s",
			e.Resource,
		)
	}

	return fmt.Sprintf(
		"saldo insuficiente fora do cofre para %s",
		e.Resource,
	)
}

type VaultState struct {
	GroupJID string

	JID string

	TierID string

	Gold int

	MagicCrystals int

	StellarStones int
}

type VaultSnapshot struct {
	Vault VaultState

	Tier VaultTier

	WalletGold int

	WalletMagicCrystals int

	WalletStellarStones int
}

type VaultTransferResult struct {
	Action string

	Resource VaultResource

	Amount int

	Snapshot VaultSnapshot
}

func VaultTiers() []VaultTier {
	result :=
		make(
			[]VaultTier,
			len(vaultTiers),
		)

	copy(
		result,
		vaultTiers,
	)

	return result
}

func VaultTierByID(
	id string,
) (VaultTier, bool) {
	for _, tier := range vaultTiers {

		if tier.ID == id {
			return tier, true
		}
	}

	return VaultTier{}, false
}

func ParseVaultResource(
	value string,
) (VaultResource, bool) {
	switch strings.ToLower(
		strings.TrimSpace(value),
	) {
	case "gold",
		"ouro":
		return VaultResourceGold,
			true

	case "cristal",
		"cristais",
		"crystal",
		"crystals":
		return VaultResourceMagicCrystals,
			true

	case "pedra",
		"pedras",
		"estelar",
		"estelares",
		"pedraestelar",
		"pedrasestelares":
		return VaultResourceStellarStones,
			true

	default:
		return "",
			false
	}
}

func (r VaultResource) Name() string {
	switch r {
	case VaultResourceGold:
		return "Gold"

	case VaultResourceMagicCrystals:
		return "Cristais Mágicos"

	case VaultResourceStellarStones:
		return "Pedras Estelares"

	default:
		return "Recurso"
	}
}

func (r VaultResource) Icon() string {
	switch r {
	case VaultResourceGold:
		return "💰"

	case VaultResourceMagicCrystals:
		return "💎"

	case VaultResourceStellarStones:
		return "🌠"

	default:
		return "📦"
	}
}

func ensureVaultSchema() error {
	if database.DB == nil {
		return errors.New(
			"banco de dados não inicializado",
		)
	}

	_, err :=
		database.DB.Exec(`
			CREATE TABLE IF NOT EXISTS rpg_vaults (
				group_jid TEXT NOT NULL,
				jid TEXT NOT NULL,

				vault_type TEXT NOT NULL
					DEFAULT 'IRON_350K',

				gold INTEGER NOT NULL
					DEFAULT 0
					CHECK (gold >= 0),

				magic_crystals INTEGER NOT NULL
					DEFAULT 0
					CHECK (magic_crystals >= 0),

				stellar_stones INTEGER NOT NULL
					DEFAULT 0
					CHECK (stellar_stones >= 0),

				created_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,

				updated_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,

				PRIMARY KEY (
					group_jid,
					jid
				),

				FOREIGN KEY (
					group_jid,
					jid
				)
					REFERENCES rpg_players (
						group_jid,
						jid
					)
					ON DELETE CASCADE
			);
		`)

	if err != nil {
		return fmt.Errorf(
			"erro criando tabela de cofres: %w",
			err,
		)
	}

	_, err =
		database.DB.Exec(`
			CREATE TABLE IF NOT EXISTS rpg_vault_transactions (
				id INTEGER PRIMARY KEY AUTOINCREMENT,

				group_jid TEXT NOT NULL,

				jid TEXT NOT NULL,

				resource TEXT NOT NULL,

				amount INTEGER NOT NULL
					CHECK (amount > 0),

				direction TEXT NOT NULL
					CHECK (
						direction IN (
							'DEPOSIT',
							'WITHDRAW'
						)
					),

				vault_type TEXT NOT NULL,

				created_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,

				FOREIGN KEY (
					group_jid,
					jid
				)
					REFERENCES rpg_vaults (
						group_jid,
						jid
					)
					ON DELETE CASCADE
			);
		`)

	if err != nil {
		return fmt.Errorf(
			"erro criando histórico do cofre: %w",
			err,
		)
	}

	_, err =
		database.DB.Exec(`
			CREATE INDEX IF NOT EXISTS
				idx_rpg_vault_transactions_player
			ON rpg_vault_transactions (
				group_jid,
				jid,
				created_at
			);
		`)

	if err != nil {
		return fmt.Errorf(
			"erro criando índice do cofre: %w",
			err,
		)
	}

	return nil
}

func ensureVaultTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
) (*VaultState, error) {
	var exists int

	err :=
		tx.QueryRow(`
			SELECT 1
			FROM rpg_players
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&exists,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			ErrWalletNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro validando jogador RPG para cofre: %w",
				err,
			)
	}

	_, err =
		tx.Exec(`
			INSERT INTO rpg_vaults (
				group_jid,
				jid,
				vault_type
			)
			VALUES (?, ?, ?)

			ON CONFLICT (
				group_jid,
				jid
			)
			DO NOTHING
		`,
			groupJID,
			jid,
			DefaultVaultTier,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro criando cofre: %w",
				err,
			)
	}

	var vault VaultState

	err =
		tx.QueryRow(`
			SELECT
				group_jid,
				jid,
				vault_type,
				gold,
				magic_crystals,
				stellar_stones
			FROM rpg_vaults
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&vault.GroupJID,
			&vault.JID,
			&vault.TierID,
			&vault.Gold,
			&vault.MagicCrystals,
			&vault.StellarStones,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando cofre: %w",
				err,
			)
	}

	return &vault, nil
}

func vaultStoredBalance(
	vault *VaultState,
	resource VaultResource,
) (int, error) {
	switch resource {
	case VaultResourceGold:
		return vault.Gold, nil

	case VaultResourceMagicCrystals:
		return vault.MagicCrystals, nil

	case VaultResourceStellarStones:
		return vault.StellarStones, nil

	default:
		return 0,
			ErrVaultInvalidResource
	}
}

func vaultColumn(
	resource VaultResource,
) (string, error) {
	switch resource {
	case VaultResourceGold:
		return "gold", nil

	case VaultResourceMagicCrystals:
		return "magic_crystals", nil

	case VaultResourceStellarStones:
		return "stellar_stones", nil

	default:
		return "",
			ErrVaultInvalidResource
	}
}

func sourceBalanceTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	resource VaultResource,
) (int, error) {
	var balance int

	var err error

	switch resource {
	case VaultResourceGold:
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

	case VaultResourceMagicCrystals:
		err =
			tx.QueryRow(`
				SELECT magic_crystals
				FROM rpg_players
				WHERE group_jid = ?
				  AND jid = ?
			`,
				groupJID,
				jid,
			).Scan(
				&balance,
			)

	case VaultResourceStellarStones:
		err =
			tx.QueryRow(`
				SELECT stellar_stones
				FROM rpg_players
				WHERE group_jid = ?
				  AND jid = ?
			`,
				groupJID,
				jid,
			).Scan(
				&balance,
			)

	default:
		return 0,
			ErrVaultInvalidResource
	}

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return 0,
			ErrWalletNotFound
	}

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro consultando recurso para cofre: %w",
				err,
			)
	}

	return balance, nil
}

func changeSourceBalanceTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	resource VaultResource,
	delta int,
) error {
	switch resource {
	case VaultResourceGold:
		_, err :=
			tx.Exec(`
				UPDATE group_wallets
				SET
					gold = gold + ?,
					updated_at = CURRENT_TIMESTAMP
				WHERE group_jid = ?
				  AND jid = ?
			`,
				delta,
				groupJID,
				jid,
			)

		return err

	case VaultResourceMagicCrystals:
		_, err :=
			tx.Exec(`
				UPDATE rpg_players
				SET
					magic_crystals =
						magic_crystals + ?,
					updated_at =
						CURRENT_TIMESTAMP
				WHERE group_jid = ?
				  AND jid = ?
			`,
				delta,
				groupJID,
				jid,
			)

		return err

	case VaultResourceStellarStones:
		_, err :=
			tx.Exec(`
				UPDATE rpg_players
				SET
					stellar_stones =
						stellar_stones + ?,
					updated_at =
						CURRENT_TIMESTAMP
				WHERE group_jid = ?
				  AND jid = ?
			`,
				delta,
				groupJID,
				jid,
			)

		return err

	default:
		return ErrVaultInvalidResource
	}
}

func snapshotVaultTx(
	tx *sql.Tx,
	vault *VaultState,
) (*VaultSnapshot, error) {
	tier, ok :=
		VaultTierByID(
			vault.TierID,
		)

	if !ok {
		return nil,
			ErrVaultUnknownTier
	}

	var walletGold int
	var magicCrystals int
	var stellarStones int

	err :=
		tx.QueryRow(`
			SELECT gold
			FROM group_wallets
			WHERE group_jid = ?
			  AND jid = ?
		`,
			vault.GroupJID,
			vault.JID,
		).Scan(
			&walletGold,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			ErrWalletNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando Gold fora do cofre: %w",
				err,
			)
	}

	err =
		tx.QueryRow(`
			SELECT
				magic_crystals,
				stellar_stones
			FROM rpg_players
			WHERE group_jid = ?
			  AND jid = ?
		`,
			vault.GroupJID,
			vault.JID,
		).Scan(
			&magicCrystals,
			&stellarStones,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando moedas RPG fora do cofre: %w",
				err,
			)
	}

	return &VaultSnapshot{
		Vault: *vault,

		Tier: tier,

		WalletGold: walletGold,

		WalletMagicCrystals: magicCrystals,

		WalletStellarStones: stellarStones,
	}, nil
}

func GetVaultSnapshot(
	groupJID string,
	jid string,
) (*VaultSnapshot, error) {
	if err :=
		ensureSchema(); err != nil {

		return nil, err
	}

	if err :=
		ensureVaultSchema(); err != nil {

		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando consulta do cofre: %w",
				err,
			)
	}

	defer tx.Rollback()

	vault, err :=
		ensureVaultTx(
			tx,
			groupJID,
			jid,
		)

	if err != nil {
		return nil, err
	}

	snapshot, err :=
		snapshotVaultTx(
			tx,
			vault,
		)

	if err != nil {
		return nil, err
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando consulta do cofre: %w",
				err,
			)
	}

	return snapshot, nil
}

func DepositVault(
	groupJID string,
	jid string,
	resource VaultResource,
	amount int,
) (*VaultTransferResult, error) {
	return transferVault(
		groupJID,
		jid,
		resource,
		amount,
		"DEPOSIT",
	)
}

func WithdrawVault(
	groupJID string,
	jid string,
	resource VaultResource,
	amount int,
) (*VaultTransferResult, error) {
	return transferVault(
		groupJID,
		jid,
		resource,
		amount,
		"WITHDRAW",
	)
}

func transferVault(
	groupJID string,
	jid string,
	resource VaultResource,
	amount int,
	direction string,
) (*VaultTransferResult, error) {
	if amount <= 0 {
		return nil,
			ErrVaultInvalidAmount
	}

	column, err :=
		vaultColumn(
			resource,
		)

	if err != nil {
		return nil, err
	}

	if direction != "DEPOSIT" &&
		direction != "WITHDRAW" {

		return nil,
			errors.New(
				"direção inválida do cofre",
			)
	}

	if err :=
		ensureSchema(); err != nil {

		return nil, err
	}

	if err :=
		ensureVaultSchema(); err != nil {

		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando movimentação do cofre: %w",
				err,
			)
	}

	defer tx.Rollback()

	vault, err :=
		ensureVaultTx(
			tx,
			groupJID,
			jid,
		)

	if err != nil {
		return nil, err
	}

	tier, ok :=
		VaultTierByID(
			vault.TierID,
		)

	if !ok {
		return nil,
			ErrVaultUnknownTier
	}

	stored, err :=
		vaultStoredBalance(
			vault,
			resource,
		)

	if err != nil {
		return nil, err
	}

	if direction == "DEPOSIT" {
		if !tier.Infinite &&
			amount >
				tier.Capacity-stored {

			return nil,
				&VaultCapacityError{
					Tier: tier,

					Resource: resource,

					Current: stored,

					Requested: amount,
				}
		}

		source, err :=
			sourceBalanceTx(
				tx,
				groupJID,
				jid,
				resource,
			)

		if err != nil {
			return nil, err
		}

		if source < amount {
			return nil,
				&VaultInsufficientError{
					Resource: resource,

					Available: source,

					Requested: amount,

					Stored: false,
				}
		}

		if err :=
			changeSourceBalanceTx(
				tx,
				groupJID,
				jid,
				resource,
				-amount,
			); err != nil {

			return nil,
				fmt.Errorf(
					"erro debitando recurso para cofre: %w",
					err,
				)
		}

		_, err =
			tx.Exec(
				fmt.Sprintf(`
					UPDATE rpg_vaults
					SET
						%s = %s + ?,
						updated_at =
							CURRENT_TIMESTAMP
					WHERE group_jid = ?
					  AND jid = ?
				`,
					column,
					column,
				),
				amount,
				groupJID,
				jid,
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro guardando recurso no cofre: %w",
					err,
				)
		}
	} else {
		if stored < amount {
			return nil,
				&VaultInsufficientError{
					Resource: resource,

					Available: stored,

					Requested: amount,

					Stored: true,
				}
		}

		_, err =
			tx.Exec(
				fmt.Sprintf(`
					UPDATE rpg_vaults
					SET
						%s = %s - ?,
						updated_at =
							CURRENT_TIMESTAMP
					WHERE group_jid = ?
					  AND jid = ?
					  AND %s >= ?
				`,
					column,
					column,
					column,
				),
				amount,
				groupJID,
				jid,
				amount,
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro retirando recurso do cofre: %w",
					err,
				)
		}

		if err :=
			changeSourceBalanceTx(
				tx,
				groupJID,
				jid,
				resource,
				amount,
			); err != nil {

			return nil,
				fmt.Errorf(
					"erro devolvendo recurso do cofre: %w",
					err,
				)
		}
	}

	_, err =
		tx.Exec(`
			INSERT INTO rpg_vault_transactions (
				group_jid,
				jid,
				resource,
				amount,
				direction,
				vault_type
			)
			VALUES (?, ?, ?, ?, ?, ?)
		`,
			groupJID,
			jid,
			string(resource),
			amount,
			direction,
			tier.ID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro registrando movimentação do cofre: %w",
				err,
			)
	}

	vault, err =
		ensureVaultTx(
			tx,
			groupJID,
			jid,
		)

	if err != nil {
		return nil, err
	}

	snapshot, err :=
		snapshotVaultTx(
			tx,
			vault,
		)

	if err != nil {
		return nil, err
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando movimentação do cofre: %w",
				err,
			)
	}

	return &VaultTransferResult{
		Action: direction,

		Resource: resource,

		Amount: amount,

		Snapshot: *snapshot,
	}, nil
}

// SetVaultTier fica disponível para a futura Loja,
// drops de boss e recompensas especiais.
//
// Não existe comando público que chame esta função.
func SetVaultTier(
	groupJID string,
	jid string,
	tierID string,
) error {
	tier, ok :=
		VaultTierByID(
			tierID,
		)

	if !ok {
		return ErrVaultUnknownTier
	}

	if err :=
		ensureSchema(); err != nil {

		return err
	}

	if err :=
		ensureVaultSchema(); err != nil {

		return err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return fmt.Errorf(
			"erro iniciando alteração do cofre: %w",
			err,
		)
	}

	defer tx.Rollback()

	vault, err :=
		ensureVaultTx(
			tx,
			groupJID,
			jid,
		)

	if err != nil {
		return err
	}

	if !tier.Infinite {
		if vault.Gold >
			tier.Capacity ||
			vault.MagicCrystals >
				tier.Capacity ||
			vault.StellarStones >
				tier.Capacity {

			return ErrVaultTierTooSmall
		}
	}

	_, err =
		tx.Exec(`
			UPDATE rpg_vaults
			SET
				vault_type = ?,
				updated_at =
					CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
		`,
			tier.ID,
			groupJID,
			jid,
		)

	if err != nil {
		return fmt.Errorf(
			"erro alterando tipo do cofre: %w",
			err,
		)
	}

	if err :=
		tx.Commit(); err != nil {

		return fmt.Errorf(
			"erro confirmando alteração do cofre: %w",
			err,
		)
	}

	return nil
}
