package rpg

import (
	"database/sql"
	"errors"
	"fmt"

	"whatsapp-sticker-bot/internal/database"
)

var (
	ErrItemNotEquipment = errors.New(
		"item não pode ser equipado",
	)

	ErrInvalidEquipmentSlot = errors.New(
		"slot de equipamento inválido",
	)

	ErrNothingEquipped = errors.New(
		"nenhum item equipado neste slot",
	)

	ErrItemEquipped = errors.New(
		"item está equipado",
	)
)

type EquippedDetail struct {
	Slot EquipmentSlot

	Item Item
}

type EquipmentSummary struct {
	Weapon *Item

	Shield *Item

	Armor *Item

	Attack int

	Defense int

	BasePower int

	PlayerLevel int

	LevelPowerBonus int

	CombatPowerBonusPercent int

	CombatPower int

	ActiveSetBonuses []ActiveSetBonus
}

func EquipItem(
	groupJID string,
	jid string,
	itemID string,
	catalog *Catalog,
) (*EquipmentSummary, error) {
	if catalog == nil {
		return nil,
			errors.New(
				"catálogo RPG não inicializado",
			)
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
		equipmentSlotForItemType(
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
				"erro iniciando equipagem: %w",
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

	var quantity int

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
			&quantity,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			ErrItemNotOwned
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando item no inventário: %w",
				err,
			)
	}

	if quantity <= 0 {
		return nil,
			ErrItemNotOwned
	}

	_, err =
		tx.Exec(`
			INSERT INTO rpg_equipment (
				group_jid,
				jid,
				slot,
				item_id
			)
			VALUES (?, ?, ?, ?)

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
			string(slot),
			item.ID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro equipando item: %w",
				err,
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando equipamento: %w",
				err,
			)
	}

	return GetEquipmentSummary(
		groupJID,
		jid,
		catalog,
	)
}

func UnequipSlot(
	groupJID string,
	jid string,
	slot EquipmentSlot,
	catalog *Catalog,
) (*EquipmentSummary, error) {
	if !slot.Valid() {
		return nil,
			ErrInvalidEquipmentSlot
	}

	if catalog == nil {
		return nil,
			errors.New(
				"catálogo RPG não inicializado",
			)
	}

	if err :=
		ensureSchema(); err != nil {

		return nil, err
	}

	result, err :=
		database.DB.Exec(`
			DELETE FROM rpg_equipment
			WHERE group_jid = ?
			  AND jid = ?
			  AND slot = ?
		`,
			groupJID,
			jid,
			string(slot),
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro desequipando item: %w",
				err,
			)
	}

	affected, err :=
		result.RowsAffected()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro validando desequipagem: %w",
				err,
			)
	}

	if affected == 0 {
		return nil,
			ErrNothingEquipped
	}

	return GetEquipmentSummary(
		groupJID,
		jid,
		catalog,
	)
}

func GetEquippedItems(
	groupJID string,
	jid string,
	catalog *Catalog,
) ([]EquippedDetail, error) {
	if catalog == nil {
		return nil,
			errors.New(
				"catálogo RPG não inicializado",
			)
	}

	if err :=
		ensureSchema(); err != nil {

		return nil, err
	}

	rows, err :=
		database.DB.Query(`
			SELECT
				slot,
				item_id
			FROM rpg_equipment
			WHERE group_jid = ?
			  AND jid = ?
			ORDER BY
				CASE slot
					WHEN 'weapon' THEN 1
					WHEN 'shield' THEN 2
					WHEN 'armor' THEN 3
					ELSE 4
				END
		`,
			groupJID,
			jid,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando equipamentos: %w",
				err,
			)
	}

	defer rows.Close()

	result :=
		make(
			[]EquippedDetail,
			0,
			3,
		)

	for rows.Next() {
		var slotText string

		var itemID string

		if err :=
			rows.Scan(
				&slotText,
				&itemID,
			); err != nil {

			return nil,
				fmt.Errorf(
					"erro lendo equipamento: %w",
					err,
				)
		}

		slot :=
			EquipmentSlot(
				slotText,
			)

		if !slot.Valid() {
			return nil,
				fmt.Errorf(
					"%w: %s",
					ErrInvalidEquipmentSlot,
					slotText,
				)
		}

		item, exists :=
			catalog.ItemByID(
				itemID,
			)

		if !exists {
			return nil,
				fmt.Errorf(
					"equipamento %s não existe no catálogo",
					itemID,
				)
		}

		expectedSlot, valid :=
			equipmentSlotForItemType(
				item.Type,
			)

		if !valid {
			return nil,
				fmt.Errorf(
					"item equipado possui tipo inválido: %s",
					item.ID,
				)
		}

		if expectedSlot != slot {
			return nil,
				fmt.Errorf(
					"item %s está no slot incorreto: %s",
					item.ID,
					slot,
				)
		}

		result =
			append(
				result,
				EquippedDetail{
					Slot: slot,

					Item: item,
				},
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"erro percorrendo equipamentos: %w",
				err,
			)
	}

	return result, nil
}

func GetEquipmentSummary(
	groupJID string,
	jid string,
	catalog *Catalog,
) (*EquipmentSummary, error) {
	equipped, err :=
		GetEquippedItems(
			groupJID,
			jid,
			catalog,
		)

	if err != nil {
		return nil, err
	}

	summary :=
		&EquipmentSummary{
			ActiveSetBonuses: make(
				[]ActiveSetBonus,
				0,
			),
		}

	equippedIDs :=
		make(
			[]string,
			0,
			len(equipped),
		)

	for _, detail := range equipped {

		item :=
			detail.Item

		summary.Attack +=
			item.Attack

		summary.Defense +=
			item.Defense

		summary.BasePower +=
			item.Power

		equippedIDs =
			append(
				equippedIDs,
				item.ID,
			)

		itemCopy :=
			item

		switch detail.Slot {

		case SlotWeapon:
			summary.Weapon =
				&itemCopy

		case SlotShield:
			summary.Shield =
				&itemCopy

		case SlotArmor:
			summary.Armor =
				&itemCopy
		}
	}

	summary.ActiveSetBonuses =
		catalog.ActiveSetBonuses(
			equippedIDs,
		)

	for _, activeBonus := range summary.ActiveSetBonuses {

		for _, effect := range activeBonus.Effects {

			if effect.Type !=
				BonusCombatPowerPercent {

				continue
			}

			summary.CombatPowerBonusPercent +=
				effect.Value
		}
	}

	summary.CombatPower =
		summary.BasePower

	if summary.CombatPowerBonusPercent > 0 {
		bonus :=
			summary.BasePower *
				summary.CombatPowerBonusPercent /
				100

		summary.CombatPower +=
			bonus
	}

	level,
		levelBonus,
		levelErr :=
		playerLevelPowerBonus(
			groupJID,
			jid,
		)

	if levelErr != nil {
		return nil,
			fmt.Errorf(
				"erro calculando PC por nível: %w",
				levelErr,
			)
	}

	summary.PlayerLevel =
		level

	summary.LevelPowerBonus =
		levelBonus

	// O bônus por nível é aditivo.
	// Cada nível acima do nível 1 concede +10 PC.
	summary.CombatPower +=
		levelBonus

	return summary, nil
}

func IsItemEquipped(
	groupJID string,
	jid string,
	itemID string,
) (bool, error) {
	if err :=
		ensureSchema(); err != nil {

		return false, err
	}

	var count int

	err :=
		database.DB.QueryRow(`
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
			&count,
		)

	if err != nil {
		return false,
			fmt.Errorf(
				"erro consultando item equipado: %w",
				err,
			)
	}

	return count > 0, nil
}

func equipmentSlotForItemType(
	itemType ItemType,
) (EquipmentSlot, bool) {
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
