package rpg

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

type RaidRewardConfig struct {
	GoldMin int
	GoldMax int

	CrystalMin int
	CrystalMax int

	StellarChancePercent int
	StellarMin           int
	StellarMax           int

	EquipmentChancePercent int
}

type RaidPlayerReward struct {
	RaidID int64

	GroupJID string

	JID  string
	Name string

	Gold int

	MagicCrystals int

	StellarStones int

	ItemID string

	ItemName string

	CreatedAt time.Time
}

func RaidRewardConfigForRarity(
	rarity Rarity,
) (
	RaidRewardConfig,
	bool,
) {
	switch rarity {

	case RarityLegendary:

		return RaidRewardConfig{
				GoldMin: 1200000,
				GoldMax: 2000000,

				CrystalMin: 100,
				CrystalMax: 160,

				StellarChancePercent: 8,
				StellarMin:           1,
				StellarMax:           1,

				EquipmentChancePercent: 12,
			},
			true

	case RarityMythic:

		return RaidRewardConfig{
				GoldMin: 2500000,
				GoldMax: 4000000,

				CrystalMin: 220,
				CrystalMax: 350,

				StellarChancePercent: 20,
				StellarMin:           1,
				StellarMax:           2,

				EquipmentChancePercent: 18,
			},
			true

	case RaritySacred:

		return RaidRewardConfig{
				GoldMin: 6000000,
				GoldMax: 10000000,

				CrystalMin: 500,
				CrystalMax: 800,

				StellarChancePercent: 45,
				StellarMin:           2,
				StellarMax:           3,

				EquipmentChancePercent: 25,
			},
			true
	}

	return RaidRewardConfig{},
		false
}

func ensureRaidRewardSchema() error {
	if err :=
		ensureRaidResolutionSchema(); err != nil {

		return err
	}

	statement := `
		CREATE TABLE IF NOT EXISTS rpg_raid_rewards (
			raid_id INTEGER NOT NULL,

			group_jid TEXT NOT NULL,

			jid TEXT NOT NULL,

			name TEXT NOT NULL
				DEFAULT '',

			gold_reward INTEGER NOT NULL
				DEFAULT 0
				CHECK (gold_reward >= 0),

			crystal_reward INTEGER NOT NULL
				DEFAULT 0
				CHECK (crystal_reward >= 0),

			stellar_stone_reward INTEGER NOT NULL
				DEFAULT 0
				CHECK (stellar_stone_reward >= 0),

			item_id TEXT NOT NULL
				DEFAULT '',

			item_name TEXT NOT NULL
				DEFAULT '',

			created_at DATETIME NOT NULL
				DEFAULT CURRENT_TIMESTAMP,

			PRIMARY KEY (
				raid_id,
				jid
			),

			FOREIGN KEY (
				raid_id
			)
				REFERENCES rpg_raid_sessions (
					id
				)
				ON DELETE CASCADE
		);
	`

	if _, err :=
		database.DB.Exec(
			statement,
		); err != nil {

		return fmt.Errorf(
			"erro criando schema de recompensas Raid: %w",
			err,
		)
	}

	return nil
}

func PendingRaidRewardIDs(
	limit int,
) (
	[]int64,
	error,
) {
	if limit <= 0 {
		limit = 20
	}

	if err :=
		ensureRaidRewardSchema(); err != nil {

		return nil,
			err
	}

	rows, err :=
		database.DB.Query(
			`
			SELECT DISTINCT
				result.raid_id
			FROM rpg_raid_results AS result
			JOIN rpg_raid_participants AS participant
			  ON participant.raid_id =
					result.raid_id
			LEFT JOIN rpg_raid_rewards AS reward
			  ON reward.raid_id =
					result.raid_id
			 AND reward.jid =
					participant.jid
			WHERE result.won = 1
			  AND reward.jid IS NULL
			ORDER BY result.raid_id ASC
			LIMIT ?
			`,
			limit,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando recompensas pendentes: %w",
				err,
			)
	}

	defer rows.Close()

	ids :=
		make(
			[]int64,
			0,
		)

	for rows.Next() {
		var raidID int64

		if err :=
			rows.Scan(
				&raidID,
			); err != nil {

			return nil,
				fmt.Errorf(
					"erro lendo Raid com recompensa pendente: %w",
					err,
				)
		}

		ids =
			append(
				ids,
				raidID,
			)
	}

	if err := rows.Err(); err != nil {
		return nil,
			err
	}

	return ids,
		nil
}

func ApplyRaidRewards(
	raidID int64,
	catalog *Catalog,
) (
	[]RaidPlayerReward,
	error,
) {
	if raidID <= 0 {
		return nil,
			ErrRaidSessionNotFound
	}

	if catalog == nil {
		return nil,
			errors.New(
				"catálogo RPG não inicializado",
			)
	}

	if err :=
		ensureRaidRewardSchema(); err != nil {

		return nil,
			err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando recompensas da Raid: %w",
				err,
			)
	}

	defer tx.Rollback()

	var (
		groupJID string

		bossID string

		won bool
	)

	err =
		tx.QueryRow(
			`
			SELECT
				group_jid,
				boss_id,
				won
			FROM rpg_raid_results
			WHERE raid_id = ?
			`,
			raidID,
		).Scan(
			&groupJID,
			&bossID,
			&won,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			ErrRaidSessionNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando resultado da Raid: %w",
				err,
			)
	}

	if !won {
		return []RaidPlayerReward{},
			nil
	}

	boss, exists :=
		RaidBossByID(
			bossID,
		)

	if !exists {
		return nil,
			fmt.Errorf(
				"%w: %s",
				ErrRaidBossNotFound,
				bossID,
			)
	}

	config, exists :=
		RaidRewardConfigForRarity(
			boss.Rarity,
		)

	if !exists {
		return nil,
			fmt.Errorf(
				"raridade sem configuração de recompensa: %s",
				boss.Rarity,
			)
	}

	rows, err :=
		tx.Query(
			`
			SELECT
				jid,
				name,
				crystal_reward_percent,
				stellar_stone_chance_percent,
				drop_chance_percent,
				rare_drop_chance_percent
			FROM rpg_raid_participants
			WHERE raid_id = ?
			ORDER BY joined_at ASC, jid ASC
			`,
			raidID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando participantes para recompensa: %w",
				err,
			)
	}

	type participantRewardSnapshot struct {
		JID string

		Name string

		CrystalBonus int

		StellarBonus int

		DropBonus int

		RareDropBonus int
	}

	participants :=
		make(
			[]participantRewardSnapshot,
			0,
			RaidMaxParticipants,
		)

	for rows.Next() {
		var participant participantRewardSnapshot

		if err :=
			rows.Scan(
				&participant.JID,
				&participant.Name,
				&participant.CrystalBonus,
				&participant.StellarBonus,
				&participant.DropBonus,
				&participant.RareDropBonus,
			); err != nil {

			rows.Close()

			return nil,
				fmt.Errorf(
					"erro lendo participante para recompensa: %w",
					err,
				)
		}

		participants =
			append(
				participants,
				participant,
			)
	}

	if err := rows.Close(); err != nil {
		return nil,
			err
	}

	if err := rows.Err(); err != nil {
		return nil,
			err
	}

	rewards :=
		make(
			[]RaidPlayerReward,
			0,
			len(
				participants,
			),
		)

	for _, participant := range participants {

		var alreadyExists int

		err :=
			tx.QueryRow(
				`
				SELECT COUNT(*)
				FROM rpg_raid_rewards
				WHERE raid_id = ?
				  AND jid = ?
				`,
				raidID,
				participant.JID,
			).Scan(
				&alreadyExists,
			)

		if err != nil {
			return nil,
				err
		}

		if alreadyExists > 0 {
			continue
		}

		if _, err :=
			ensurePlayerTx(
				tx,
				groupJID,
				participant.JID,
			); err != nil {

			return nil,
				err
		}

		goldReward, err :=
			randomRaidRewardInclusive(
				config.GoldMin,
				config.GoldMax,
			)

		if err != nil {
			return nil,
				err
		}

		crystalReward, err :=
			randomRaidRewardInclusive(
				config.CrystalMin,
				config.CrystalMax,
			)

		if err != nil {
			return nil,
				err
		}

		crystalReward =
			applyRaidRewardPercent(
				crystalReward,
				participant.CrystalBonus,
			)

		stellarReward := 0

		stellarChance :=
			clampRaidPercent(
				config.StellarChancePercent +
					participant.StellarBonus,
			)

		stellarDropped, err :=
			rollRaidRewardPercent(
				stellarChance,
			)

		if err != nil {
			return nil,
				err
		}

		if stellarDropped {
			stellarReward, err =
				randomRaidRewardInclusive(
					config.StellarMin,
					config.StellarMax,
				)

			if err != nil {
				return nil,
					err
			}
		}

		item :=
			raidEquipmentReward(
				catalog,
				boss,
			)

		itemID := ""
		itemName := ""

		if item != nil {
			dropChance :=
				clampRaidPercent(
					config.EquipmentChancePercent +
						participant.DropBonus +
						participant.RareDropBonus,
				)

			dropped, err :=
				rollRaidRewardPercent(
					dropChance,
				)

			if err != nil {
				return nil,
					err
			}

			if dropped {
				itemID =
					item.ID

				itemName =
					item.Name
			}
		}

		result, err :=
			tx.Exec(
				`
				UPDATE group_wallets
				SET
					gold = gold + ?,
					updated_at =
						CURRENT_TIMESTAMP
				WHERE group_jid = ?
				  AND jid = ?
				`,
				goldReward,
				groupJID,
				participant.JID,
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro creditando Gold da Raid: %w",
					err,
				)
		}

		affected, err :=
			result.RowsAffected()

		if err != nil {
			return nil,
				err
		}

		if affected != 1 {
			return nil,
				ErrWalletNotFound
		}

		_, err =
			tx.Exec(
				`
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
				participant.JID,
				goldReward,
				"RPG_RAID_REWARD",
				fmt.Sprintf(
					"Recompensa Raid: %s",
					boss.Name,
				),
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro registrando Gold da Raid: %w",
					err,
				)
		}

		if crystalReward > 0 {
			_, err =
				tx.Exec(
					`
					UPDATE rpg_players
					SET
						magic_crystals =
							magic_crystals + ?,
						updated_at =
							CURRENT_TIMESTAMP
					WHERE group_jid = ?
					  AND jid = ?
					`,
					crystalReward,
					groupJID,
					participant.JID,
				)

			if err != nil {
				return nil,
					fmt.Errorf(
						"erro creditando Cristais da Raid: %w",
						err,
					)
			}
		}

		if stellarReward > 0 {
			_, err =
				tx.Exec(
					`
					UPDATE rpg_players
					SET
						stellar_stones =
							stellar_stones + ?,
						updated_at =
							CURRENT_TIMESTAMP
					WHERE group_jid = ?
					  AND jid = ?
					`,
					stellarReward,
					groupJID,
					participant.JID,
				)

			if err != nil {
				return nil,
					fmt.Errorf(
						"erro creditando Pedras Estelares da Raid: %w",
						err,
					)
			}
		}

		if itemID != "" {
			_, err =
				tx.Exec(
					`
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
					participant.JID,
					itemID,
				)

			if err != nil {
				return nil,
					fmt.Errorf(
						"erro adicionando drop da Raid: %w",
						err,
					)
			}
		}

		_, err =
			tx.Exec(
				`
				INSERT INTO rpg_raid_rewards (
					raid_id,
					group_jid,
					jid,
					name,
					gold_reward,
					crystal_reward,
					stellar_stone_reward,
					item_id,
					item_name
				)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				`,
				raidID,
				groupJID,
				participant.JID,
				participant.Name,
				goldReward,
				crystalReward,
				stellarReward,
				itemID,
				itemName,
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro persistindo recompensa da Raid: %w",
					err,
				)
		}

		rewards =
			append(
				rewards,
				RaidPlayerReward{
					RaidID: raidID,

					GroupJID: groupJID,

					JID: participant.JID,

					Name: participant.Name,

					Gold: goldReward,

					MagicCrystals: crystalReward,

					StellarStones: stellarReward,

					ItemID: itemID,

					ItemName: itemName,

					CreatedAt: time.Now(),
				},
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando recompensas da Raid: %w",
				err,
			)
	}

	return rewards,
		nil
}

func RaidRewardsForRaid(
	raidID int64,
) (
	[]RaidPlayerReward,
	error,
) {
	if err :=
		ensureRaidRewardSchema(); err != nil {

		return nil,
			err
	}

	rows, err :=
		database.DB.Query(
			`
			SELECT
				raid_id,
				group_jid,
				jid,
				name,
				gold_reward,
				crystal_reward,
				stellar_stone_reward,
				item_id,
				item_name,
				created_at
			FROM rpg_raid_rewards
			WHERE raid_id = ?
			ORDER BY created_at ASC, jid ASC
			`,
			raidID,
		)

	if err != nil {
		return nil,
			err
	}

	defer rows.Close()

	rewards :=
		make(
			[]RaidPlayerReward,
			0,
			RaidMaxParticipants,
		)

	for rows.Next() {
		var reward RaidPlayerReward

		if err :=
			rows.Scan(
				&reward.RaidID,
				&reward.GroupJID,
				&reward.JID,
				&reward.Name,
				&reward.Gold,
				&reward.MagicCrystals,
				&reward.StellarStones,
				&reward.ItemID,
				&reward.ItemName,
				&reward.CreatedAt,
			); err != nil {

			return nil,
				err
		}

		rewards =
			append(
				rewards,
				reward,
			)
	}

	return rewards,
		rows.Err()
}

func raidEquipmentReward(
	catalog *Catalog,
	boss RaidBoss,
) *Item {
	setID :=
		strings.TrimSpace(
			boss.SetID,
		)

	if setID == "" {
		return nil
	}

	set, exists :=
		catalog.SetByID(
			setID,
		)

	if !exists ||
		len(
			set.Pieces,
		) == 0 {

		return nil
	}

	index, err :=
		randomRaidRewardInclusive(
			0,
			len(
				set.Pieces,
			)-1,
		)

	if err != nil {
		return nil
	}

	item, exists :=
		catalog.ItemByID(
			set.Pieces[index],
		)

	if !exists {
		return nil
	}

	return &item
}

func applyRaidRewardPercent(
	value int,
	bonus int,
) int {
	if value <= 0 {
		return 0
	}

	if bonus <= 0 {
		return value
	}

	return value *
		(100 + bonus) /
		100
}

func clampRaidPercent(
	value int,
) int {
	if value < 0 {
		return 0
	}

	if value > 100 {
		return 100
	}

	return value
}

func rollRaidRewardPercent(
	chance int,
) (
	bool,
	error,
) {
	chance =
		clampRaidPercent(
			chance,
		)

	if chance <= 0 {
		return false,
			nil
	}

	if chance >= 100 {
		return true,
			nil
	}

	roll, err :=
		randomRaidRewardInclusive(
			1,
			100,
		)

	if err != nil {
		return false,
			err
	}

	return roll <= chance,
		nil
}

func randomRaidRewardInclusive(
	minimum int,
	maximum int,
) (
	int,
	error,
) {
	if maximum < minimum {
		return 0,
			fmt.Errorf(
				"intervalo inválido: %d-%d",
				minimum,
				maximum,
			)
	}

	if minimum == maximum {
		return minimum,
			nil
	}

	size :=
		maximum -
			minimum +
			1

	value, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(
				int64(
					size,
				),
			),
		)

	if err != nil {
		return 0,
			err
	}

	return minimum +
			int(
				value.Int64(),
			),
		nil
}
