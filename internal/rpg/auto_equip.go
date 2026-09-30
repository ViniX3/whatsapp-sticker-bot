package rpg

import (
	"database/sql"
	"errors"
	"fmt"

	"whatsapp-sticker-bot/internal/database"
)

type AutoEquipResult struct {
	Equipped bool

	Slot EquipmentSlot

	Item Item

	HadPrevious bool

	PreviousItem Item
}

func autoEquipSlot(
	itemType ItemType,
) (
	EquipmentSlot,
	bool,
) {
	switch itemType {
	case ItemTypeWeapon:
		return SlotWeapon,
			true

	case ItemTypeShield:
		return SlotShield,
			true

	case ItemTypeArmor:
		return SlotArmor,
			true
	}

	return "",
		false
}

// AutoEquipIfBetter equipa o item somente quando:
//
//   - o slot está vazio; ou
//   - o novo item possui Power maior.
//
// Em empate, o equipamento atual é mantido.
func AutoEquipIfBetter(
	groupJID string,
	jid string,
	itemID string,
	catalog *Catalog,
) (
	*AutoEquipResult,
	error,
) {
	if catalog == nil {
		return nil,
			ErrItemNotFound
	}

	item, exists :=
		catalog.ItemByID(
			itemID,
		)

	if !exists {
		return nil,
			ErrItemNotFound
	}

	slot, valid :=
		autoEquipSlot(
			item.Type,
		)

	if !valid {
		return nil,
			ErrItemNotEquipment
	}

	if err :=
		ensureSchema(); err != nil {

		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando auto-equip: %w",
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

		return nil, err
	}

	var owned int

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
			item.ID,
		).Scan(
			&owned,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) ||
		owned <= 0 {

		return nil,
			ErrItemNotOwned
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro verificando item para auto-equip: %w",
				err,
			)
	}

	result :=
		&AutoEquipResult{
			Slot: slot,

			Item: item,
		}

	var currentItemID string

	err =
		tx.QueryRow(`
			SELECT item_id
			FROM rpg_equipment
			WHERE group_jid = ?
			  AND jid = ?
			  AND slot = ?
		`,
			groupJID,
			jid,
			string(
				slot,
			),
		).Scan(
			&currentItemID,
		)

	switch {
	case errors.Is(
		err,
		sql.ErrNoRows,
	):
		// Slot vazio. Novo item será equipado.

	case err != nil:
		return nil,
			fmt.Errorf(
				"erro consultando equipamento atual: %w",
				err,
			)

	default:
		current, exists :=
			catalog.ItemByID(
				currentItemID,
			)

		if exists {
			result.HadPrevious =
				true

			result.PreviousItem =
				current

			// Só substitui quando for realmente maior.
			if item.Power <=
				current.Power {

				if err :=
					tx.Commit(); err != nil {

					return nil,
						fmt.Errorf(
							"erro finalizando comparação de auto-equip: %w",
							err,
						)
				}

				return result,
					nil
			}
		}
	}

	_, err =
		tx.Exec(`
			INSERT INTO rpg_equipment (
				group_jid,
				jid,
				slot,
				item_id,
				equipped_at
			)
			VALUES (
				?,
				?,
				?,
				?,
				CURRENT_TIMESTAMP
			)

			ON CONFLICT (
				group_jid,
				jid,
				slot
			)
			DO UPDATE SET
				item_id =
					excluded.item_id,

				equipped_at =
					CURRENT_TIMESTAMP
		`,
			groupJID,
			jid,
			string(
				slot,
			),
			item.ID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro realizando auto-equip: %w",
				err,
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando auto-equip: %w",
				err,
			)
	}

	result.Equipped =
		true

	return result, nil
}
