package rpg

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"whatsapp-sticker-bot/internal/database"
)

var (
	ErrWalletNotFound = errors.New(
		"carteira gold não encontrada",
	)

	ErrInvalidAmount = errors.New(
		"valor inválido",
	)

	ErrInsufficientMagicCrystals = errors.New(
		"cristais mágicos insuficientes",
	)

	ErrInsufficientStellarStones = errors.New(
		"pedras estelares insuficientes",
	)

	ErrInvalidItemID = errors.New(
		"id de item inválido",
	)

	ErrInvalidItemQuantity = errors.New(
		"quantidade de item inválida",
	)

	ErrItemNotOwned = errors.New(
		"item não pertence ao jogador",
	)
)

func Init() error {
	return ensureSchema()
}

func ensureSchema() error {
	if database.DB == nil {
		return errors.New(
			"banco de dados não inicializado",
		)
	}

	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS rpg_players (
			group_jid TEXT NOT NULL,
			jid TEXT NOT NULL,

			magic_crystals INTEGER NOT NULL DEFAULT 0
				CHECK (magic_crystals >= 0),

			stellar_stones INTEGER NOT NULL DEFAULT 0
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
				REFERENCES group_wallets (
					group_jid,
					jid
				)
				ON DELETE CASCADE
		);
		`,

		`
		CREATE TABLE IF NOT EXISTS rpg_inventory (
			group_jid TEXT NOT NULL,
			jid TEXT NOT NULL,

			item_id TEXT NOT NULL,

			quantity INTEGER NOT NULL DEFAULT 1
				CHECK (quantity > 0),

			acquired_at DATETIME NOT NULL
				DEFAULT CURRENT_TIMESTAMP,

			updated_at DATETIME NOT NULL
				DEFAULT CURRENT_TIMESTAMP,

			PRIMARY KEY (
				group_jid,
				jid,
				item_id
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
		`,

		`
		CREATE TABLE IF NOT EXISTS rpg_equipment (
			group_jid TEXT NOT NULL,
			jid TEXT NOT NULL,

			slot TEXT NOT NULL
				CHECK (
					slot IN (
						'weapon',
						'shield',
						'armor'
					)
				),

			item_id TEXT NOT NULL,

			equipped_at DATETIME NOT NULL
				DEFAULT CURRENT_TIMESTAMP,

			PRIMARY KEY (
				group_jid,
				jid,
				slot
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
		`,

		`
		CREATE INDEX IF NOT EXISTS
			idx_rpg_inventory_item
		ON rpg_inventory (
			item_id
		);
		`,

		`
		CREATE INDEX IF NOT EXISTS
			idx_rpg_equipment_item
		ON rpg_equipment (
			item_id
		);
		`,
	}

	for _, statement := range statements {

		_, err :=
			database.DB.Exec(
				statement,
			)

		if err != nil {
			return fmt.Errorf(
				"erro inicializando schema RPG: %w",
				err,
			)
		}
	}

	return nil
}

func GetOrCreatePlayer(
	groupJID string,
	jid string,
) (*PlayerState, error) {
	if err := ensureSchema(); err != nil {
		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando jogador RPG: %w",
				err,
			)
	}

	defer tx.Rollback()

	player, err :=
		ensurePlayerTx(
			tx,
			groupJID,
			jid,
		)

	if err != nil {
		return nil, err
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando jogador RPG: %w",
				err,
			)
	}

	return player, nil
}

func GetPlayer(
	groupJID string,
	jid string,
) (*PlayerState, error) {
	return GetOrCreatePlayer(
		groupJID,
		jid,
	)
}

func AddMagicCrystals(
	groupJID string,
	jid string,
	amount int,
) (int, error) {
	if amount <= 0 {
		return 0,
			ErrInvalidAmount
	}

	if err := ensureSchema(); err != nil {
		return 0, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro iniciando crédito de cristais: %w",
				err,
			)
	}

	defer tx.Rollback()

	if _, err :=
		ensurePlayerTx(
			tx,
			groupJID,
			jid,
		); err != nil {

		return 0, err
	}

	_, err =
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
			amount,
			groupJID,
			jid,
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro creditando cristais mágicos: %w",
				err,
			)
	}

	var balance int

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

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro consultando cristais mágicos: %w",
				err,
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return 0,
			fmt.Errorf(
				"erro confirmando cristais mágicos: %w",
				err,
			)
	}

	return balance, nil
}

func SpendMagicCrystals(
	groupJID string,
	jid string,
	amount int,
) (int, error) {
	if amount <= 0 {
		return 0,
			ErrInvalidAmount
	}

	if err := ensureSchema(); err != nil {
		return 0, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro iniciando débito de cristais: %w",
				err,
			)
	}

	defer tx.Rollback()

	if _, err :=
		ensurePlayerTx(
			tx,
			groupJID,
			jid,
		); err != nil {

		return 0, err
	}

	result, err :=
		tx.Exec(`
			UPDATE rpg_players
			SET
				magic_crystals =
					magic_crystals - ?,

				updated_at =
					CURRENT_TIMESTAMP

			WHERE group_jid = ?
			  AND jid = ?
			  AND magic_crystals >= ?
		`,
			amount,
			groupJID,
			jid,
			amount,
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro debitando cristais mágicos: %w",
				err,
			)
	}

	rowsAffected, err :=
		result.RowsAffected()

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro validando débito de cristais: %w",
				err,
			)
	}

	if rowsAffected != 1 {
		return 0,
			ErrInsufficientMagicCrystals
	}

	var balance int

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

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro consultando saldo de cristais: %w",
				err,
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return 0,
			fmt.Errorf(
				"erro confirmando débito de cristais: %w",
				err,
			)
	}

	return balance, nil
}

func AddStellarStones(
	groupJID string,
	jid string,
	amount int,
) (int, error) {
	if amount <= 0 {
		return 0,
			ErrInvalidAmount
	}

	if err := ensureSchema(); err != nil {
		return 0, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro iniciando crédito de pedras estelares: %w",
				err,
			)
	}

	defer tx.Rollback()

	if _, err :=
		ensurePlayerTx(
			tx,
			groupJID,
			jid,
		); err != nil {

		return 0, err
	}

	_, err =
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
			amount,
			groupJID,
			jid,
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro creditando pedras estelares: %w",
				err,
			)
	}

	var balance int

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

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro consultando pedras estelares: %w",
				err,
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return 0,
			fmt.Errorf(
				"erro confirmando pedras estelares: %w",
				err,
			)
	}

	return balance, nil
}

func SpendStellarStones(
	groupJID string,
	jid string,
	amount int,
) (int, error) {
	if amount <= 0 {
		return 0,
			ErrInvalidAmount
	}

	if err := ensureSchema(); err != nil {
		return 0, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro iniciando débito de pedras estelares: %w",
				err,
			)
	}

	defer tx.Rollback()

	if _, err :=
		ensurePlayerTx(
			tx,
			groupJID,
			jid,
		); err != nil {

		return 0, err
	}

	result, err :=
		tx.Exec(`
			UPDATE rpg_players
			SET
				stellar_stones =
					stellar_stones - ?,

				updated_at =
					CURRENT_TIMESTAMP

			WHERE group_jid = ?
			  AND jid = ?
			  AND stellar_stones >= ?
		`,
			amount,
			groupJID,
			jid,
			amount,
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro debitando pedras estelares: %w",
				err,
			)
	}

	rowsAffected, err :=
		result.RowsAffected()

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro validando débito de pedras estelares: %w",
				err,
			)
	}

	if rowsAffected != 1 {
		return 0,
			ErrInsufficientStellarStones
	}

	var balance int

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

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro consultando saldo de pedras estelares: %w",
				err,
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return 0,
			fmt.Errorf(
				"erro confirmando débito de pedras estelares: %w",
				err,
			)
	}

	return balance, nil
}

func AddInventoryItem(
	groupJID string,
	jid string,
	itemID string,
	quantity int,
) error {
	itemID =
		strings.TrimSpace(
			itemID,
		)

	if itemID == "" {
		return ErrInvalidItemID
	}

	if quantity <= 0 {
		return ErrInvalidItemQuantity
	}

	if err := ensureSchema(); err != nil {
		return err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return fmt.Errorf(
			"erro iniciando adição ao inventário: %w",
			err,
		)
	}

	defer tx.Rollback()

	if _, err :=
		ensurePlayerTx(
			tx,
			groupJID,
			jid,
		); err != nil {

		return err
	}

	_, err =
		tx.Exec(`
			INSERT INTO rpg_inventory (
				group_jid,
				jid,
				item_id,
				quantity
			)
			VALUES (?, ?, ?, ?)

			ON CONFLICT (
				group_jid,
				jid,
				item_id
			)
			DO UPDATE SET
				quantity =
					rpg_inventory.quantity
					+ excluded.quantity,

				updated_at =
					CURRENT_TIMESTAMP
		`,
			groupJID,
			jid,
			itemID,
			quantity,
		)

	if err != nil {
		return fmt.Errorf(
			"erro adicionando item ao inventário: %w",
			err,
		)
	}

	if err :=
		tx.Commit(); err != nil {

		return fmt.Errorf(
			"erro confirmando item no inventário: %w",
			err,
		)
	}

	return nil
}

func RemoveInventoryItem(
	groupJID string,
	jid string,
	itemID string,
	quantity int,
) error {
	itemID =
		strings.TrimSpace(
			itemID,
		)

	if itemID == "" {
		return ErrInvalidItemID
	}

	if quantity <= 0 {
		return ErrInvalidItemQuantity
	}

	if err := ensureSchema(); err != nil {
		return err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return fmt.Errorf(
			"erro iniciando remoção do inventário: %w",
			err,
		)
	}

	defer tx.Rollback()

	var currentQuantity int

	err =
		tx.QueryRow(`
			SELECT quantity
			FROM rpg_inventory
			WHERE group_jid = ?
			  AND jid = ?
			  AND item_id = ?
		`,
			groupJID,
			jid,
			itemID,
		).Scan(
			&currentQuantity,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return ErrItemNotOwned
	}

	if err != nil {
		return fmt.Errorf(
			"erro consultando item no inventário: %w",
			err,
		)
	}

	if currentQuantity < quantity {
		return ErrItemNotOwned
	}

	newQuantity :=
		currentQuantity -
			quantity

	if newQuantity == 0 {
		var equippedCount int

		err =
			tx.QueryRow(`
				SELECT COUNT(*)
				FROM rpg_equipment
				WHERE group_jid = ?
				  AND jid = ?
				  AND item_id = ?
			`,
				groupJID,
				jid,
				itemID,
			).Scan(
				&equippedCount,
			)

		if err != nil {
			return fmt.Errorf(
				"erro verificando item equipado: %w",
				err,
			)
		}

		if equippedCount > 0 {
			return fmt.Errorf(
				"%w: não é possível remover a última unidade de um item equipado",
				ErrItemEquipped,
			)
		}
	}

	if newQuantity == 0 {
		_, err =
			tx.Exec(`
				DELETE FROM rpg_inventory
				WHERE group_jid = ?
				  AND jid = ?
				  AND item_id = ?
			`,
				groupJID,
				jid,
				itemID,
			)
	} else {
		_, err =
			tx.Exec(`
				UPDATE rpg_inventory
				SET
					quantity = ?,
					updated_at =
						CURRENT_TIMESTAMP
				WHERE group_jid = ?
				  AND jid = ?
				  AND item_id = ?
			`,
				newQuantity,
				groupJID,
				jid,
				itemID,
			)
	}

	if err != nil {
		return fmt.Errorf(
			"erro removendo item do inventário: %w",
			err,
		)
	}

	if err :=
		tx.Commit(); err != nil {

		return fmt.Errorf(
			"erro confirmando remoção do inventário: %w",
			err,
		)
	}

	return nil
}

func GetInventory(
	groupJID string,
	jid string,
) ([]InventoryEntry, error) {
	if err := ensureSchema(); err != nil {
		return nil, err
	}

	rows, err :=
		database.DB.Query(`
			SELECT
				item_id,
				quantity
			FROM rpg_inventory
			WHERE group_jid = ?
			  AND jid = ?
			ORDER BY acquired_at ASC
		`,
			groupJID,
			jid,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando inventário: %w",
				err,
			)
	}

	defer rows.Close()

	inventory :=
		make(
			[]InventoryEntry,
			0,
		)

	for rows.Next() {
		var entry InventoryEntry

		if err :=
			rows.Scan(
				&entry.ItemID,
				&entry.Quantity,
			); err != nil {

			return nil,
				fmt.Errorf(
					"erro lendo inventário: %w",
					err,
				)
		}

		inventory =
			append(
				inventory,
				entry,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"erro percorrendo inventário: %w",
				err,
			)
	}

	return inventory, nil
}

func GetInventoryQuantity(
	groupJID string,
	jid string,
	itemID string,
) (int, error) {
	itemID =
		strings.TrimSpace(
			itemID,
		)

	if itemID == "" {
		return 0,
			ErrInvalidItemID
	}

	if err := ensureSchema(); err != nil {
		return 0, err
	}

	var quantity int

	err :=
		database.DB.QueryRow(`
			SELECT quantity
			FROM rpg_inventory
			WHERE group_jid = ?
			  AND jid = ?
			  AND item_id = ?
		`,
			groupJID,
			jid,
			itemID,
		).Scan(
			&quantity,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return 0, nil
	}

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro consultando quantidade do item: %w",
				err,
			)
	}

	return quantity, nil
}

func ensurePlayerTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
) (*PlayerState, error) {
	var walletExists int

	err :=
		tx.QueryRow(`
			SELECT 1
			FROM group_wallets
			WHERE group_jid = ?
			  AND jid = ?
			LIMIT 1
		`,
			groupJID,
			jid,
		).Scan(
			&walletExists,
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
				"erro validando carteira para RPG: %w",
				err,
			)
	}

	_, err =
		tx.Exec(`
			INSERT INTO rpg_players (
				group_jid,
				jid
			)
			VALUES (?, ?)

			ON CONFLICT (
				group_jid,
				jid
			)
			DO NOTHING
		`,
			groupJID,
			jid,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro criando jogador RPG: %w",
				err,
			)
	}

	player :=
		&PlayerState{}

	err =
		tx.QueryRow(`
			SELECT
				group_jid,
				jid,
				magic_crystals,
				stellar_stones

			FROM rpg_players

			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&player.GroupJID,
			&player.JID,
			&player.MagicCrystals,
			&player.StellarStones,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando jogador RPG: %w",
				err,
			)
	}

	return player, nil
}
