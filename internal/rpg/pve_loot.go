package rpg

import (
	"database/sql"
	"fmt"
)

type PVEMaterialDrop struct {
	Material Material

	Quantity int
}

func rollPVEMaterialDrop(
	enemy Enemy,
	catalog *MaterialCatalog,
) (
	*PVEMaterialDrop,
	error,
) {
	chance := 0
	minQuantity := 0
	maxQuantity := 0

	var targetRarity Rarity

	switch enemy.Rarity {
	case RarityCommon:
		chance = 40
		targetRarity = RarityCommon
		minQuantity = 1
		maxQuantity = 1

	case RarityRare:
		chance = 70
		targetRarity = RarityRare
		minQuantity = 1
		maxQuantity = 2

	case RarityEpic:
		chance = 100
		targetRarity = RarityEpic
		minQuantity = 2
		maxQuantity = 4

	case RarityLegendary:
		chance = 100
		targetRarity = RarityLegendary
		minQuantity = 4
		maxQuantity = 8

	default:
		return nil, nil
	}

	drop, err :=
		randomPVEPercentRoll(
			chance,
		)

	if err != nil {
		return nil, err
	}

	if !drop {
		return nil, nil
	}

	actualRarity, err :=
		resolveGatheringRarity(
			enemy.Region,
			targetRarity,
		)

	if err != nil {
		return nil, err
	}

	material, err :=
		randomGatheringMaterial(
			enemy.Region,
			actualRarity,
			catalog,
		)

	if err != nil {
		return nil, err
	}

	quantity, err :=
		randomPVEInclusive(
			minQuantity,
			maxQuantity,
		)

	if err != nil {
		return nil, err
	}

	return &PVEMaterialDrop{
		Material: material,
		Quantity: quantity,
	}, nil
}

func rollPVEEquipmentDrop(
	enemy Enemy,
	catalog *Catalog,
) (
	*Item,
	error,
) {
	chance := 0

	switch enemy.Rarity {
	case RarityRare:
		chance = 3

	case RarityEpic:
		chance = 6

	case RarityLegendary:
		chance = 5

	default:
		return nil, nil
	}

	drop, err :=
		randomPVEPercentRoll(
			chance,
		)

	if err != nil {
		return nil, err
	}

	if !drop {
		return nil, nil
	}

	pool :=
		make(
			[]Item,
			0,
		)

	for _, item := range catalog.Items() {

		if !pveEquipmentEligibleForEnemy(
			enemy,
			item,
		) {

			continue
		}

		pool =
			append(
				pool,
				item,
			)
	}

	if len(pool) == 0 {
		return nil, nil
	}

	index, err :=
		randomPVEInt(
			len(pool),
		)

	if err != nil {
		return nil, err
	}

	item := pool[index]

	return &item, nil
}

func creditPVELootTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	materialDrop *PVEMaterialDrop,
	equipmentDrop *Item,
) error {
	if materialDrop != nil {
		_, err :=
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
				materialDrop.Material.ID,
				materialDrop.Quantity,
			)

		if err != nil {
			return fmt.Errorf(
				"erro adicionando material PvE ao inventário: %w",
				err,
			)
		}
	}

	if equipmentDrop != nil {
		_, err :=
			tx.Exec(`
				INSERT INTO rpg_inventory (
					group_jid,
					jid,
					item_id,
					quantity
				)
				VALUES (?, ?, ?, 1)

				ON CONFLICT (
					group_jid,
					jid,
					item_id
				)
				DO UPDATE SET
					quantity =
						rpg_inventory.quantity + 1,

					updated_at =
						CURRENT_TIMESTAMP
			`,
				groupJID,
				jid,
				equipmentDrop.ID,
			)

		if err != nil {
			return fmt.Errorf(
				"erro adicionando equipamento PvE ao inventário: %w",
				err,
			)
		}
	}

	return nil
}
