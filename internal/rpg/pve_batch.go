package rpg

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

const (
	PVEBatchMaxBattles            = 1000
	PVEBatchLossPenaltyMinSeconds = 1
	PVEBatchLossPenaltyMaxSeconds = 3
	PVEBatchRecoveryCap           = 5 * time.Minute
)

var (
	ErrInvalidPVEBatchAmount = errors.New("quantidade de batalhas PvE em lote inválida")
	pveBatchMutex            sync.Mutex
)

type PVEBatchCooldownError struct {
	Remaining time.Duration
	ReadyAt   time.Time
}

func (err *PVEBatchCooldownError) Error() string {
	return fmt.Sprintf("expedição PvE em recuperação por %s", err.Remaining)
}

type PVEBatchMaterialDrop struct {
	Material Material
	Quantity int
}

type PVEBatchEquipmentDrop struct {
	Item     Item
	Quantity int
}

type PVEBatchResult struct {
	Battles        int
	Wins           int
	Losses         int
	BossEncounters int
	BossKills      int
	PlayerPower    int
	FixedRegion    GatheringRegion
	GoldReward     int
	CrystalReward  int
	Materials      []PVEBatchMaterialDrop
	Equipment      []PVEBatchEquipmentDrop
	Recovery       time.Duration
	ReadyAt        time.Time

	// Permite ao handler reutilizar exatamente a regra atual de XP.
	VictoryResults []PVEResult
}

func PlayPVEBatch(
	groupJID string,
	jid string,
	region GatheringRegion,
	battles int,
	itemCatalog *Catalog,
	materialCatalog *MaterialCatalog,
	enemyCatalog *EnemyCatalog,
) (*PVEBatchResult, error) {
	if battles < 1 || battles > PVEBatchMaxBattles {
		return nil, ErrInvalidPVEBatchAmount
	}
	if itemCatalog == nil {
		return nil, errors.New("catálogo de equipamentos não carregado")
	}
	if materialCatalog == nil {
		return nil, errors.New("catálogo de materiais não carregado")
	}
	if enemyCatalog == nil {
		return nil, errors.New("catálogo de monstros não carregado")
	}
	if region != "" && !region.Valid() {
		return nil, ErrInvalidPVERegion
	}
	if err := ensureSchema(); err != nil {
		return nil, err
	}
	if err := ensurePVESchema(); err != nil {
		return nil, err
	}
	if err := ensurePVEBatchSchema(); err != nil {
		return nil, err
	}

	// Uma expedição por vez no processo evita duas execuções simultâneas
	// concedendo recompensas antes de o cooldown ser persistido.
	pveBatchMutex.Lock()
	defer pveBatchMutex.Unlock()

	now := time.Now().UTC().Truncate(time.Second)

	remaining, readyAt, err := getPVEBatchRemainingCooldown(groupJID, jid, now)
	if err != nil {
		return nil, err
	}
	if remaining > 0 {
		return nil, &PVEBatchCooldownError{
			Remaining: remaining,
			ReadyAt:   readyAt,
		}
	}

	summary, err := GetEquipmentSummary(groupJID, jid, itemCatalog)
	if err != nil {
		return nil, fmt.Errorf("erro consultando PC do jogador para PvE em lote: %w", err)
	}
	if summary.CombatPower <= 0 {
		return nil, ErrInvalidPVEPlayerPower
	}

	result := &PVEBatchResult{
		Battles:        battles,
		PlayerPower:    summary.CombatPower,
		FixedRegion:    region,
		Materials:      make([]PVEBatchMaterialDrop, 0),
		Equipment:      make([]PVEBatchEquipmentDrop, 0),
		VictoryResults: make([]PVEResult, 0, battles),
	}

	materials := make(map[string]PVEBatchMaterialDrop)
	equipment := make(map[string]PVEBatchEquipmentDrop)
	bossKills := make([]Enemy, 0)
	recovery := time.Duration(0)

	materialRollInterval :=
		pveBatchMaterialRollInterval(
			battles,
		)

	for battle := 0; battle < battles; battle++ {
		battleRegion := region
		if battleRegion == "" {
			battleRegion, err = randomPVERegion()
			if err != nil {
				return nil, err
			}
		}

		rarity, err := randomPVERarity()
		if err != nil {
			return nil, err
		}

		enemy, err := randomPVEEnemy(enemyCatalog, battleRegion, rarity)
		if err != nil {
			return nil, err
		}

		if enemy.Boss {
			result.BossEncounters++
		}

		winChance := PVEWinChance(result.PlayerPower, enemy.Power, enemy.Boss)
		won, err := randomPVEPercentRoll(winChance)
		if err != nil {
			return nil, err
		}

		if !won {
			result.Losses++

			if recovery < PVEBatchRecoveryCap {
				seconds, err := randomPVEInclusive(
					PVEBatchLossPenaltyMinSeconds,
					PVEBatchLossPenaltyMaxSeconds,
				)
				if err != nil {
					return nil, err
				}
				recovery = addPVEBatchPenalty(recovery, seconds)
			}
			continue
		}

		result.Wins++

		goldReward, err := randomPVEInclusive(enemy.GoldMin, enemy.GoldMax)
		if err != nil {
			return nil, err
		}
		result.GoldReward += goldReward

		crystals, err := rollPVECrystals(enemy, summary)
		if err != nil {
			return nil, err
		}
		result.CrystalReward += crystals

		var materialDrop *PVEMaterialDrop

		if result.Wins%materialRollInterval == 0 {
			materialDrop, err =
				rollPVEMaterialDrop(
					enemy,
					materialCatalog,
				)

			if err != nil {
				return nil, err
			}

			if materialDrop != nil {
				entry := materials[materialDrop.Material.ID]
				entry.Material = materialDrop.Material
				entry.Quantity += materialDrop.Quantity
				materials[materialDrop.Material.ID] = entry
			}
		}

		equipmentDrop, err := rollPVEEquipmentDrop(enemy, itemCatalog)
		if err != nil {
			return nil, err
		}
		if equipmentDrop != nil {
			entry := equipment[equipmentDrop.ID]
			entry.Item = *equipmentDrop
			entry.Quantity++
			equipment[equipmentDrop.ID] = entry
		}

		if enemy.Boss {
			result.BossKills++
			bossKills = append(bossKills, enemy)
		}

		victory := PVEResult{
			Region:           battleRegion,
			Enemy:            enemy,
			PlayerPower:      result.PlayerPower,
			EnemyPower:       enemy.Power,
			WinChancePercent: winChance,
			Won:              true,
			GoldReward:       goldReward,
			CrystalReward:    crystals,
			MaterialDrop:     materialDrop,
			EquipmentDrop:    equipmentDrop,
			BossKill:         enemy.Boss,
			BattledAt:        now,
			NextAvailableAt:  now.Add(PVECooldown),
		}

		if enemy.Boss && enemy.Blessing != nil {
			blessingCopy := *enemy.Blessing
			victory.Blessing = &blessingCopy
		}

		result.VictoryResults = append(result.VictoryResults, victory)
	}

	result.Recovery = recovery
	result.ReadyAt = now.Add(recovery)

	for _, drop := range materials {
		result.Materials = append(result.Materials, drop)
	}
	sort.Slice(result.Materials, func(i, j int) bool {
		ri := result.Materials[i].Material.Rarity.Rank()
		rj := result.Materials[j].Material.Rarity.Rank()
		if ri != rj {
			return ri > rj
		}
		return result.Materials[i].Material.Name < result.Materials[j].Material.Name
	})

	for _, drop := range equipment {
		result.Equipment = append(result.Equipment, drop)
	}
	sort.Slice(result.Equipment, func(i, j int) bool {
		ri := result.Equipment[i].Item.Rarity.Rank()
		rj := result.Equipment[j].Item.Rarity.Rank()
		if ri != rj {
			return ri > rj
		}
		if result.Equipment[i].Item.Power != result.Equipment[j].Item.Power {
			return result.Equipment[i].Item.Power > result.Equipment[j].Item.Power
		}
		return result.Equipment[i].Item.Name < result.Equipment[j].Item.Name
	})

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("erro iniciando transação do PvE em lote: %w", err)
	}
	defer tx.Rollback()

	if _, err := ensurePlayerTx(tx, groupJID, jid); err != nil {
		return nil, err
	}
	if err := ensurePVEPlayerTx(tx, groupJID, jid); err != nil {
		return nil, err
	}
	if err := creditPVEBatchRewardsTx(tx, groupJID, jid, result); err != nil {
		return nil, err
	}
	if err := creditPVEBatchLootTx(tx, groupJID, jid, result); err != nil {
		return nil, err
	}

	for _, enemy := range bossKills {
		if err := registerPVEBossKillTx(tx, groupJID, jid, enemy, now); err != nil {
			return nil, err
		}
	}

	if err := updatePVEBatchStatsTx(tx, groupJID, jid, result, now); err != nil {
		return nil, err
	}
	if err := setPVEBatchReadyAtTx(tx, groupJID, jid, result.ReadyAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("erro confirmando PvE em lote: %w", err)
	}

	return result, nil
}

func pveBatchMaterialRollInterval(
	battles int,
) int {
	switch {
	case battles <= 10:
		return 1

	case battles <= 50:
		return 2

	case battles <= 100:
		return 4

	case battles <= 250:
		return 6

	case battles <= 500:
		return 10

	default:
		return 20
	}
}

func addPVEBatchPenalty(current time.Duration, seconds int) time.Duration {
	if seconds <= 0 {
		return current
	}

	next := current + time.Duration(seconds)*time.Second
	if next > PVEBatchRecoveryCap {
		return PVEBatchRecoveryCap
	}
	return next
}

func creditPVEBatchRewardsTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	result *PVEBatchResult,
) error {
	if result.GoldReward > 0 {
		updateResult, err := tx.Exec(`
			UPDATE group_wallets
			SET gold = gold + ?,
			    updated_at = CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
		`,
			result.GoldReward,
			groupJID,
			jid,
		)
		if err != nil {
			return fmt.Errorf("erro creditando Gold do PvE em lote: %w", err)
		}

		rows, err := updateResult.RowsAffected()
		if err != nil {
			return fmt.Errorf("erro validando crédito de Gold do PvE em lote: %w", err)
		}
		if rows != 1 {
			return ErrWalletNotFound
		}

		_, err = tx.Exec(`
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
			jid,
			result.GoldReward,
			"RPG_PVE_BATCH_REWARD",
			fmt.Sprintf(
				"Recompensa PvE em lote: %d batalhas, %d vitórias",
				result.Battles,
				result.Wins,
			),
		)
		if err != nil {
			return fmt.Errorf("erro registrando Gold do PvE em lote: %w", err)
		}
	}

	if result.CrystalReward > 0 {
		_, err := tx.Exec(`
			UPDATE rpg_players
			SET magic_crystals = magic_crystals + ?,
			    updated_at = CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
		`,
			result.CrystalReward,
			groupJID,
			jid,
		)
		if err != nil {
			return fmt.Errorf("erro creditando Cristais do PvE em lote: %w", err)
		}
	}

	return nil
}

func creditPVEBatchLootTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	result *PVEBatchResult,
) error {
	for _, drop := range result.Materials {
		if drop.Quantity <= 0 {
			continue
		}

		_, err := tx.Exec(`
			INSERT INTO rpg_inventory (
				group_jid,
				jid,
				item_id,
				quantity
			)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (group_jid, jid, item_id)
			DO UPDATE SET
				quantity = rpg_inventory.quantity + excluded.quantity,
				updated_at = CURRENT_TIMESTAMP
		`,
			groupJID,
			jid,
			drop.Material.ID,
			drop.Quantity,
		)
		if err != nil {
			return fmt.Errorf(
				"erro creditando material do PvE em lote %s: %w",
				drop.Material.ID,
				err,
			)
		}
	}

	for _, drop := range result.Equipment {
		if drop.Quantity <= 0 {
			continue
		}

		_, err := tx.Exec(`
			INSERT INTO rpg_inventory (
				group_jid,
				jid,
				item_id,
				quantity
			)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (group_jid, jid, item_id)
			DO UPDATE SET
				quantity = rpg_inventory.quantity + excluded.quantity,
				updated_at = CURRENT_TIMESTAMP
		`,
			groupJID,
			jid,
			drop.Item.ID,
			drop.Quantity,
		)
		if err != nil {
			return fmt.Errorf(
				"erro creditando equipamento do PvE em lote %s: %w",
				drop.Item.ID,
				err,
			)
		}
	}

	return nil
}

func updatePVEBatchStatsTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	result *PVEBatchResult,
	now time.Time,
) error {
	_, err := tx.Exec(`
		UPDATE rpg_pve
		SET
			last_pve_at = ?,
			total_battles = total_battles + ?,
			wins = wins + ?,
			losses = losses + ?,
			boss_kills = boss_kills + ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
	`,
		now,
		result.Battles,
		result.Wins,
		result.Losses,
		result.BossKills,
		groupJID,
		jid,
	)
	if err != nil {
		return fmt.Errorf("erro atualizando estatísticas do PvE em lote: %w", err)
	}
	return nil
}

func getPVEBatchRemainingCooldown(
	groupJID string,
	jid string,
	now time.Time,
) (time.Duration, time.Time, error) {
	var readyAt time.Time

	err := database.DB.QueryRow(`
		SELECT ready_at
		FROM rpg_pve_batch_cooldowns
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	).Scan(&readyAt)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, time.Time{}, nil
	}
	if err != nil {
		return 0, time.Time{}, fmt.Errorf(
			"erro consultando recuperação do PvE em lote: %w",
			err,
		)
	}
	if !now.Before(readyAt) {
		return 0, readyAt, nil
	}

	return readyAt.Sub(now), readyAt, nil
}

func setPVEBatchReadyAtTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	readyAt time.Time,
) error {
	_, err := tx.Exec(`
		INSERT INTO rpg_pve_batch_cooldowns (
			group_jid,
			jid,
			ready_at
		)
		VALUES (?, ?, ?)
		ON CONFLICT (group_jid, jid)
		DO UPDATE SET
			ready_at = excluded.ready_at,
			updated_at = CURRENT_TIMESTAMP
	`,
		groupJID,
		jid,
		readyAt,
	)
	if err != nil {
		return fmt.Errorf("erro registrando recuperação do PvE em lote: %w", err)
	}
	return nil
}

func ensurePVEBatchSchema() error {
	_, err := database.DB.Exec(`
		CREATE TABLE IF NOT EXISTS rpg_pve_batch_cooldowns (
			group_jid TEXT NOT NULL,
			jid TEXT NOT NULL,
			ready_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (group_jid, jid)
		);
	`)
	if err != nil {
		return fmt.Errorf("erro garantindo schema do PvE em lote: %w", err)
	}
	return nil
}
