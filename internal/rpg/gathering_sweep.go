package rpg

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

const (
	GatheringSweepRollsPerRegion = 2

	GatheringSweepBonusRollChancePercent = 30
)

type GatheringSweepRegionResult struct {
	Region GatheringRegion

	Drops []GatheringDrop

	Rolls int

	BonusRoll bool

	TotalItems int

	EpicItems int

	LegendaryItems int
}

type GatheringSweepResult struct {
	Regions []GatheringSweepRegionResult

	Rolls int

	BonusRegions int

	TotalItems int

	EpicItems int

	LegendaryItems int

	GatheredAt time.Time

	NextAvailableAt time.Time
}

var gatheringSweepRegions = []GatheringRegion{
	GatheringForest,
	GatheringQuarry,
	GatheringMine,
}

func GatherAllRegions(
	groupJID string,
	jid string,
	catalog *MaterialCatalog,
) (
	*GatheringSweepResult,
	error,
) {
	if catalog == nil {
		return nil,
			errors.New(
				"catálogo de materiais não inicializado",
			)
	}

	if err :=
		ValidateGatheringPools(
			catalog,
		); err != nil {

		return nil, err
	}

	if err := ensureSchema(); err != nil {
		return nil, err
	}

	if err :=
		ensureGatheringSchema(); err != nil {

		return nil, err
	}

	if err :=
		ensureGatheringRegionSchema(); err != nil {

		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando varredura de coleta: %w",
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

	_, err =
		tx.Exec(`
			INSERT INTO rpg_gathering (
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
				"erro criando estado da varredura: %w",
				err,
			)
	}

	var lastGathered sql.NullTime

	err =
		tx.QueryRow(`
			SELECT last_gathered_at
			FROM rpg_gathering
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&lastGathered,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando cooldown da varredura: %w",
				err,
			)
	}

	now :=
		time.Now().
			UTC().
			Truncate(time.Second)

	if lastGathered.Valid {
		nextAvailable :=
			lastGathered.Time.
				Add(
					GatheringCooldown,
				)

		if now.Before(
			nextAvailable,
		) {
			return nil,
				&GatheringCooldownError{
					Remaining: nextAvailable.
						Sub(now),

					NextAvailableAt: nextAvailable,
				}
		}
	}

	result :=
		&GatheringSweepResult{
			Regions: make(
				[]GatheringSweepRegionResult,
				0,
				len(gatheringSweepRegions),
			),

			GatheredAt: now,

			NextAvailableAt: now.Add(
				GatheringCooldown,
			),
		}

	for _, region := range gatheringSweepRegions {

		regionResult, err :=
			rollGatheringSweepRegion(
				region,
				catalog,
			)

		if err != nil {
			return nil, err
		}

		for _, drop := range regionResult.Drops {

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
					drop.Material.ID,
					drop.Quantity,
				)

			if err != nil {
				return nil,
					fmt.Errorf(
						"erro adicionando material %s da varredura ao inventário: %w",
						drop.Material.ID,
						err,
					)
			}
		}

		result.Regions =
			append(
				result.Regions,
				*regionResult,
			)

		result.Rolls +=
			regionResult.Rolls

		result.TotalItems +=
			regionResult.TotalItems

		result.EpicItems +=
			regionResult.EpicItems

		result.LegendaryItems +=
			regionResult.LegendaryItems

		if regionResult.BonusRoll {
			result.BonusRegions++
		}
	}

	if result.TotalItems <= 0 {
		return nil,
			ErrNoGatheringMaterials
	}

	_, err =
		tx.Exec(`
			UPDATE rpg_gathering
			SET
				last_gathered_at = ?,

				total_gatherings =
					total_gatherings + 1,

				total_resources =
					total_resources + ?,

				epic_resources =
					epic_resources + ?,

				legendary_resources =
					legendary_resources + ?,

				updated_at =
					CURRENT_TIMESTAMP

			WHERE group_jid = ?
			  AND jid = ?
		`,
			now,
			result.TotalItems,
			result.EpicItems,
			result.LegendaryItems,
			groupJID,
			jid,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro atualizando estatísticas da varredura: %w",
				err,
			)
	}

	for _, region := range gatheringSweepRegions {

		_, err =
			tx.Exec(`
				INSERT INTO rpg_gathering_regions (
					group_jid,
					jid,
					region,
					last_gathered_at
				)
				VALUES (?, ?, ?, ?)

				ON CONFLICT (
					group_jid,
					jid,
					region
				)
				DO UPDATE SET
					last_gathered_at =
						excluded.last_gathered_at,

					updated_at =
						CURRENT_TIMESTAMP
			`,
				groupJID,
				jid,
				string(region),
				now,
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro atualizando cooldown da região %s: %w",
					region,
					err,
				)
		}
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando varredura de coleta: %w",
				err,
			)
	}

	return result,
		nil
}

func rollGatheringSweepRegion(
	region GatheringRegion,
	catalog *MaterialCatalog,
) (
	*GatheringSweepRegionResult,
	error,
) {
	if !region.Valid() {
		return nil,
			ErrInvalidGatheringRegion
	}

	bonusRoll, err :=
		gatheringChance(
			GatheringSweepBonusRollChancePercent,
		)

	if err != nil {
		return nil, err
	}

	rolls :=
		GatheringSweepRollsPerRegion

	if bonusRoll {
		rolls++
	}

	result :=
		&GatheringSweepRegionResult{
			Region: region,

			Rolls: rolls,

			BonusRoll: bonusRoll,

			Drops: make(
				[]GatheringDrop,
				0,
				rolls,
			),
		}

	dropIndex :=
		make(
			map[string]int,
		)

	for roll := 0; roll < rolls; roll++ {
		rolledRarity, err :=
			rollGatheringRarity()

		if err != nil {
			return nil, err
		}

		actualRarity, err :=
			resolveGatheringRarity(
				region,
				rolledRarity,
			)

		if err != nil {
			return nil, err
		}

		material, err :=
			randomGatheringMaterial(
				region,
				actualRarity,
				catalog,
			)

		if err != nil {
			return nil, err
		}

		quantity, err :=
			randomGatheringQuantity(
				material.Rarity,
			)

		if err != nil {
			return nil, err
		}

		if index, exists :=
			dropIndex[material.ID]; exists {

			result.Drops[index].Quantity +=
				quantity
		} else {
			dropIndex[material.ID] =
				len(result.Drops)

			result.Drops =
				append(
					result.Drops,
					GatheringDrop{
						Material: material,

						Quantity: quantity,
					},
				)
		}

		result.TotalItems +=
			quantity

		switch material.Rarity {
		case RarityEpic:
			result.EpicItems +=
				quantity

		case RarityLegendary:
			result.LegendaryItems +=
				quantity
		}
	}

	return result,
		nil
}
