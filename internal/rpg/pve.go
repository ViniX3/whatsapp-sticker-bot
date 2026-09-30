package rpg

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

const (
	PVECooldown = 10 * time.Second

	PVECommonChancePercent = 74.5
	PVERareChancePercent   = 20.0
	PVEEpicChancePercent   = 5.0
	PVEBossChancePercent   = 0.5

	PVEMinWinChancePercent = 2
	PVEMaxWinChancePercent = 98

	PVEBossMinWinChancePercent = 1
	PVEBossMaxWinChancePercent = 80
)

var (
	ErrInvalidPVERegion = errors.New(
		"região PvE inválida",
	)

	ErrEmptyEnemyPool = errors.New(
		"nenhum monstro disponível para o encontro",
	)

	ErrInvalidPVEPlayerPower = errors.New(
		"poder de combate inválido para PvE",
	)
)

type PVECooldownError struct {
	Remaining       time.Duration
	NextAvailableAt time.Time
}

func (
	err *PVECooldownError,
) Error() string {
	return fmt.Sprintf(
		"PvE em cooldown por %s",
		err.Remaining,
	)
}

type PVEResult struct {
	Region GatheringRegion

	Enemy Enemy

	PlayerPower int

	EnemyPower int

	WinChancePercent int

	Won bool

	GoldReward int

	CrystalReward int

	MaterialDrop *PVEMaterialDrop

	EquipmentDrop *Item

	BossKill bool

	Blessing *BossBlessing

	BattledAt time.Time

	NextAvailableAt time.Time
}

type PVEStats struct {
	TotalBattles int

	Wins int

	Losses int

	BossKills int

	LastBattleAt sql.NullTime
}

func InitPVE() error {
	if err := ensureSchema(); err != nil {
		return err
	}

	return ensurePVESchema()
}

func PlayPVE(
	groupJID string,
	jid string,
	region GatheringRegion,
	itemCatalog *Catalog,
	materialCatalog *MaterialCatalog,
	enemyCatalog *EnemyCatalog,
) (
	*PVEResult,
	error,
) {
	if itemCatalog == nil {
		return nil,
			errors.New(
				"catálogo de equipamentos não carregado",
			)
	}

	if enemyCatalog == nil {
		return nil,
			errors.New(
				"catálogo de monstros não carregado",
			)
	}

	if materialCatalog == nil {
		return nil,
			errors.New(
				"catálogo de materiais não carregado",
			)
	}

	if err := ensureSchema(); err != nil {
		return nil, err
	}

	if err := ensurePVESchema(); err != nil {
		return nil, err
	}

	if region == "" {
		randomRegion, err :=
			randomPVERegion()

		if err != nil {
			return nil, err
		}

		region = randomRegion
	}

	if !region.Valid() {
		return nil,
			ErrInvalidPVERegion
	}

	summary, err :=
		GetEquipmentSummary(
			groupJID,
			jid,
			itemCatalog,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando PC do jogador para PvE: %w",
				err,
			)
	}

	playerPower :=
		summary.CombatPower

	if playerPower <= 0 {
		return nil,
			ErrInvalidPVEPlayerPower
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando PvE: %w",
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

	if err :=
		ensurePVEPlayerTx(
			tx,
			groupJID,
			jid,
		); err != nil {

		return nil, err
	}

	now :=
		time.Now().
			UTC().
			Truncate(
				time.Second,
			)

	rarity, err :=
		randomPVERarity()

	if err != nil {
		return nil, err
	}

	enemy, err :=
		randomPVEEnemy(
			enemyCatalog,
			region,
			rarity,
		)

	if err != nil {
		return nil, err
	}

	winChance :=
		PVEWinChance(
			playerPower,
			enemy.Power,
			enemy.Boss,
		)

	won, err :=
		randomPVEPercentRoll(
			winChance,
		)

	if err != nil {
		return nil, err
	}

	result :=
		&PVEResult{
			Region: region,

			Enemy: enemy,

			PlayerPower: playerPower,

			EnemyPower: enemy.Power,

			WinChancePercent: winChance,

			Won: won,

			BattledAt: now,

			NextAvailableAt: now.Add(
				PVECooldown,
			),
		}

	if won {
		goldReward, err :=
			randomPVEInclusive(
				enemy.GoldMin,
				enemy.GoldMax,
			)

		if err != nil {
			return nil, err
		}

		result.GoldReward =
			goldReward

		crystals, err :=
			rollPVECrystals(
				enemy,
				summary,
			)

		if err != nil {
			return nil, err
		}

		result.CrystalReward =
			crystals

		materialDrop, err :=
			rollPVEMaterialDrop(
				enemy,
				materialCatalog,
			)

		if err != nil {
			return nil, err
		}

		result.MaterialDrop =
			materialDrop

		equipmentDrop, err :=
			rollPVEEquipmentDrop(
				enemy,
				itemCatalog,
			)

		if err != nil {
			return nil, err
		}

		result.EquipmentDrop =
			equipmentDrop

		if err :=
			creditPVERewardsTx(
				tx,
				groupJID,
				jid,
				enemy,
				goldReward,
				crystals,
			); err != nil {

			return nil, err
		}

		if err :=
			creditPVELootTx(
				tx,
				groupJID,
				jid,
				materialDrop,
				equipmentDrop,
			); err != nil {

			return nil, err
		}

		if enemy.Boss {
			result.BossKill = true

			if enemy.Blessing != nil {
				blessingCopy :=
					*enemy.Blessing

				result.Blessing =
					&blessingCopy
			}

			if err :=
				registerPVEBossKillTx(
					tx,
					groupJID,
					jid,
					enemy,
					now,
				); err != nil {

				return nil, err
			}
		}
	}

	if err :=
		updatePVEStatsTx(
			tx,
			groupJID,
			jid,
			won,
			enemy.Boss &&
				won,
			now,
		); err != nil {

		return nil, err
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando PvE: %w",
				err,
			)
	}

	return result, nil
}

func PVEWinChance(
	playerPower int,
	enemyPower int,
	boss bool,
) int {
	if playerPower <= 0 {
		return 0
	}

	if enemyPower <= 0 {
		if boss {
			return PVEBossMaxWinChancePercent
		}

		return PVEMaxWinChancePercent
	}

	// ========================================================
	// CURVA DE PODER PvE
	// ========================================================
	//
	// Relação aproximada entre PC do jogador e PC do inimigo:
	//
	// 0,50x ->  5%
	// 0,75x -> 30%
	// 1,00x -> 50%
	// 1,25x -> 65%
	// 1,50x -> 80%
	// 2,00x -> 95%
	// 3,00x -> 98%
	//
	// Isso faz com que equipamentos melhores tenham impacto
	// perceptível no PvE, sem remover completamente o risco.
	//
	// Bosses recebem penalidade adicional de 15 pontos e
	// continuam limitados a no máximo 80% de vitória.
	// ========================================================

	ratio :=
		float64(playerPower) /
			float64(enemyPower)

	chance := 50

	switch {

	// --------------------------------------------------------
	// Jogador extremamente superior
	// --------------------------------------------------------

	case ratio >= 3.0:
		chance = 98

	// --------------------------------------------------------
	// 2x até 3x
	//
	// 2,00x -> 95%
	// 3,00x -> 98%
	// --------------------------------------------------------

	case ratio >= 2.0:
		chance =
			95 +
				int(
					math.Round(
						(ratio-2.0)*3.0,
					),
				)

	// --------------------------------------------------------
	// 1,5x até 2x
	//
	// 1,50x -> 80%
	// 2,00x -> 95%
	// --------------------------------------------------------

	case ratio >= 1.5:
		chance =
			80 +
				int(
					math.Round(
						(ratio-1.5)*30.0,
					),
				)

	// --------------------------------------------------------
	// 1x até 1,5x
	//
	// 1,00x -> 50%
	// 1,25x -> 65%
	// 1,50x -> 80%
	// --------------------------------------------------------

	case ratio >= 1.0:
		chance =
			50 +
				int(
					math.Round(
						(ratio-1.0)*60.0,
					),
				)

	// --------------------------------------------------------
	// 0,75x até 1x
	//
	// 0,75x -> 30%
	// 1,00x -> 50%
	// --------------------------------------------------------

	case ratio >= 0.75:
		chance =
			30 +
				int(
					math.Round(
						(ratio-0.75)*80.0,
					),
				)

	// --------------------------------------------------------
	// 0,5x até 0,75x
	//
	// 0,50x -> 5%
	// 0,75x -> 30%
	// --------------------------------------------------------

	case ratio >= 0.5:
		chance =
			5 +
				int(
					math.Round(
						(ratio-0.5)*100.0,
					),
				)

	// --------------------------------------------------------
	// Menos da metade do PC
	// --------------------------------------------------------

	default:
		chance =
			PVEMinWinChancePercent
	}

	// ========================================================
	// BOSS
	// ========================================================

	if boss {
		chance -= 15

		return clampPVEChance(
			chance,
			PVEBossMinWinChancePercent,
			PVEBossMaxWinChancePercent,
		)
	}

	return clampPVEChance(
		chance,
		PVEMinWinChancePercent,
		PVEMaxWinChancePercent,
	)
}

func GetPVEStats(
	groupJID string,
	jid string,
) (
	*PVEStats,
	error,
) {
	if err :=
		ensurePVESchema(); err != nil {

		return nil, err
	}

	stats :=
		&PVEStats{}

	err :=
		database.DB.QueryRow(`
			SELECT
				total_battles,
				wins,
				losses,
				boss_kills,
				last_pve_at
			FROM rpg_pve
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&stats.TotalBattles,
			&stats.Wins,
			&stats.Losses,
			&stats.BossKills,
			&stats.LastBattleAt,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return &PVEStats{},
			nil
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando estatísticas PvE: %w",
				err,
			)
	}

	return stats, nil
}

func GetPVERemainingCooldown(
	groupJID string,
	jid string,
) (
	time.Duration,
	error,
) {
	if err :=
		ensurePVESchema(); err != nil {

		return 0, err
	}

	var lastBattle sql.NullTime

	err :=
		database.DB.QueryRow(`
			SELECT last_pve_at
			FROM rpg_pve
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&lastBattle,
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
				"erro consultando cooldown PvE: %w",
				err,
			)
	}

	if !lastBattle.Valid {
		return 0, nil
	}

	next :=
		lastBattle.Time.
			Add(
				PVECooldown,
			)

	remaining :=
		time.Until(
			next,
		)

	if remaining <= 0 {
		return 0, nil
	}

	return remaining, nil
}

func randomPVERarity() (
	Rarity,
	error,
) {
	// Escala de 10.000:
	//
	// 7450 = 74,5% Common
	// 2000 = 20,0% Rare
	//  500 =  5,0% Epic
	//   50 =  0,5% Legendary Boss

	draw, err :=
		randomPVEInt(
			10000,
		)

	if err != nil {
		return "",
			fmt.Errorf(
				"erro sorteando raridade PvE: %w",
				err,
			)
	}

	switch {
	case draw < 7450:
		return RarityCommon, nil

	case draw < 9450:
		return RarityRare, nil

	case draw < 9950:
		return RarityEpic, nil

	default:
		return RarityLegendary, nil
	}
}

func randomPVEEnemy(
	catalog *EnemyCatalog,
	region GatheringRegion,
	rarity Rarity,
) (
	Enemy,
	error,
) {
	pool :=
		catalog.
			EnemiesByRegionAndRarity(
				region,
				rarity,
			)

	if len(pool) == 0 {
		return Enemy{},
			ErrEmptyEnemyPool
	}

	index, err :=
		randomPVEInt(
			len(pool),
		)

	if err != nil {
		return Enemy{},
			fmt.Errorf(
				"erro sorteando monstro PvE: %w",
				err,
			)
	}

	return pool[index], nil
}

func randomPVERegion() (
	GatheringRegion,
	error,
) {
	regions :=
		[]GatheringRegion{
			GatheringForest,
			GatheringQuarry,
			GatheringMine,
		}

	index, err :=
		randomPVEInt(
			len(regions),
		)

	if err != nil {
		return "",
			fmt.Errorf(
				"erro sorteando região PvE: %w",
				err,
			)
	}

	return regions[index], nil
}

func randomPVEPercentRoll(
	chancePercent int,
) (
	bool,
	error,
) {
	if chancePercent <= 0 {
		return false, nil
	}

	if chancePercent >= 100 {
		return true, nil
	}

	draw, err :=
		randomPVEInt(
			100,
		)

	if err != nil {
		return false,
			fmt.Errorf(
				"erro sorteando resultado PvE: %w",
				err,
			)
	}

	return draw <
			chancePercent,
		nil
}

func rollPVECrystals(
	enemy Enemy,
	summary *EquipmentSummary,
) (
	int,
	error,
) {

	// O PvE não possui mais cooldown.
	// Portanto, Cristais não são mais garantidos.
	if enemy.CrystalDropChancePercent <= 0 {
		return 0, nil
	}

	dropped, err :=
		randomPVEPercentRoll(
			enemy.CrystalDropChancePercent,
		)

	if err != nil {
		return 0, err
	}

	if !dropped {
		return 0, nil
	}

	minimum := 0
	maximum := 0

	if enemy.Boss {

		minimum = 30
		maximum = 60

	} else {

		switch enemy.Rarity {

		case RarityCommon:
			minimum = 1
			maximum = 3

		case RarityRare:
			minimum = 4
			maximum = 8

		case RarityEpic:
			minimum = 10
			maximum = 20

		case RarityLegendary:
			minimum = 20
			maximum = 35

		default:
			return 0, nil
		}
	}

	reward, err :=
		randomPVEInclusive(
			minimum,
			maximum,
		)

	if err != nil {
		return 0, err
	}

	bonusPercent :=
		pveCrystalRewardBonus(
			summary,
		)

	reward =
		applyPVECrystalRewardBonus(
			reward,
			bonusPercent,
		)

	return reward, nil
}

func pveCrystalRewardBonus(
	summary *EquipmentSummary,
) int {

	if summary == nil {
		return 0
	}

	bonus := 0

	for _, active := range summary.ActiveSetBonuses {

		for _, effect := range active.Effects {

			if effect.Type !=
				BonusCrystalRewardPercent {

				continue
			}

			bonus +=
				effect.Value
		}
	}

	return bonus
}

func applyPVECrystalRewardBonus(
	reward int,
	bonusPercent int,
) int {

	if reward <= 0 {
		return 0
	}

	if bonusPercent <= 0 {
		return reward
	}

	bonus :=
		reward *
			bonusPercent /
			100

	// Um bônus de set válido sempre deve produzir
	// pelo menos +1 Cristal.
	if bonus < 1 {
		bonus = 1
	}

	return reward +
		bonus
}

func creditPVERewardsTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	enemy Enemy,
	goldReward int,
	crystalReward int,
) error {
	if goldReward > 0 {
		result, err :=
			tx.Exec(`
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
				jid,
			)

		if err != nil {
			return fmt.Errorf(
				"erro creditando Gold do PvE: %w",
				err,
			)
		}

		rows, err :=
			result.RowsAffected()

		if err != nil {
			return fmt.Errorf(
				"erro validando crédito de Gold do PvE: %w",
				err,
			)
		}

		if rows != 1 {
			return ErrWalletNotFound
		}

		_, err =
			tx.Exec(`
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
				goldReward,
				"RPG_PVE_REWARD",
				fmt.Sprintf(
					"Recompensa PvE: %s",
					enemy.Name,
				),
			)

		if err != nil {
			return fmt.Errorf(
				"erro registrando recompensa Gold do PvE: %w",
				err,
			)
		}
	}

	if crystalReward > 0 {
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
				crystalReward,
				groupJID,
				jid,
			)

		if err != nil {
			return fmt.Errorf(
				"erro creditando Cristais do PvE: %w",
				err,
			)
		}
	}

	return nil
}

func updatePVEStatsTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	won bool,
	bossKill bool,
	now time.Time,
) error {
	winIncrement := 0
	lossIncrement := 0
	bossIncrement := 0

	if won {
		winIncrement = 1
	} else {
		lossIncrement = 1
	}

	if bossKill {
		bossIncrement = 1
	}

	_, err :=
		tx.Exec(`
			UPDATE rpg_pve
			SET
				last_pve_at = ?,
				total_battles =
					total_battles + 1,
				wins =
					wins + ?,
				losses =
					losses + ?,
				boss_kills =
					boss_kills + ?,
				updated_at =
					CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
		`,
			now,
			winIncrement,
			lossIncrement,
			bossIncrement,
			groupJID,
			jid,
		)

	if err != nil {
		return fmt.Errorf(
			"erro atualizando estatísticas PvE: %w",
			err,
		)
	}

	return nil
}

func registerPVEBossKillTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	enemy Enemy,
	now time.Time,
) error {
	_, err :=
		tx.Exec(`
			INSERT INTO rpg_pve_boss_kills (
				group_jid,
				jid,
				boss_id,
				kills,
				first_kill_at,
				last_kill_at
			)
			VALUES (?, ?, ?, 1, ?, ?)

			ON CONFLICT (
				group_jid,
				jid,
				boss_id
			)
			DO UPDATE SET
				kills =
					rpg_pve_boss_kills.kills + 1,
				last_kill_at =
					excluded.last_kill_at
		`,
			groupJID,
			jid,
			enemy.ID,
			now,
			now,
		)

	if err != nil {
		return fmt.Errorf(
			"erro registrando kill de boss PvE: %w",
			err,
		)
	}

	if enemy.Blessing == nil {
		return nil
	}

	expiresAt :=
		now.Add(
			time.Duration(
				enemy.Blessing.
					DurationSeconds,
			) * time.Second,
		)

	_, err =
		tx.Exec(`
			INSERT INTO rpg_pve_blessings (
				group_jid,
				jid,
				blessing_id,
				boss_id,
				region,
				activated_at,
				expires_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?)

			ON CONFLICT (
				group_jid,
				jid,
				blessing_id
			)
			DO UPDATE SET
				boss_id =
					excluded.boss_id,
				region =
					excluded.region,
				activated_at =
					excluded.activated_at,
				expires_at =
					excluded.expires_at
		`,
			groupJID,
			jid,
			enemy.Blessing.ID,
			enemy.ID,
			enemy.Region,
			now,
			expiresAt,
		)

	if err != nil {
		return fmt.Errorf(
			"erro registrando bênção PvE: %w",
			err,
		)
	}

	return nil
}

func ensurePVEPlayerTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
) error {
	_, err :=
		tx.Exec(`
			INSERT INTO rpg_pve (
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
		return fmt.Errorf(
			"erro inicializando PvE do jogador: %w",
			err,
		)
	}

	return nil
}

func ensurePVESchema() error {
	if database.DB == nil {
		return errors.New(
			"banco de dados não inicializado",
		)
	}

	statements :=
		[]string{
			`
			CREATE TABLE IF NOT EXISTS rpg_pve (
				group_jid TEXT NOT NULL,
				jid TEXT NOT NULL,

				last_pve_at DATETIME,

				total_battles INTEGER NOT NULL
					DEFAULT 0
					CHECK (total_battles >= 0),

				wins INTEGER NOT NULL
					DEFAULT 0
					CHECK (wins >= 0),

				losses INTEGER NOT NULL
					DEFAULT 0
					CHECK (losses >= 0),

				boss_kills INTEGER NOT NULL
					DEFAULT 0
					CHECK (boss_kills >= 0),

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
			`,

			`
			CREATE TABLE IF NOT EXISTS rpg_pve_boss_kills (
				group_jid TEXT NOT NULL,
				jid TEXT NOT NULL,
				boss_id TEXT NOT NULL,

				kills INTEGER NOT NULL
					DEFAULT 0
					CHECK (kills >= 0),

				first_kill_at DATETIME NOT NULL,
				last_kill_at DATETIME NOT NULL,

				PRIMARY KEY (
					group_jid,
					jid,
					boss_id
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
			CREATE TABLE IF NOT EXISTS rpg_pve_blessings (
				group_jid TEXT NOT NULL,
				jid TEXT NOT NULL,
				blessing_id TEXT NOT NULL,
				boss_id TEXT NOT NULL,
				region TEXT NOT NULL,

				activated_at DATETIME NOT NULL,
				expires_at DATETIME NOT NULL,

				PRIMARY KEY (
					group_jid,
					jid,
					blessing_id
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
		}

	for _, statement := range statements {

		if _, err :=
			database.DB.Exec(
				statement,
			); err != nil {

			return fmt.Errorf(
				"erro criando schema PvE: %w",
				err,
			)
		}
	}

	return nil
}

func randomPVEInclusive(
	minimum int,
	maximum int,
) (
	int,
	error,
) {
	if maximum < minimum {
		return 0,
			fmt.Errorf(
				"intervalo PvE inválido: %d-%d",
				minimum,
				maximum,
			)
	}

	if maximum == minimum {
		return minimum, nil
	}

	size :=
		maximum -
			minimum +
			1

	draw, err :=
		randomPVEInt(
			size,
		)

	if err != nil {
		return 0, err
	}

	return minimum +
			draw,
		nil
}

func randomPVEInt(
	maximum int,
) (
	int,
	error,
) {
	if maximum <= 0 {
		return 0,
			fmt.Errorf(
				"limite aleatório PvE inválido: %d",
				maximum,
			)
	}

	draw, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(
				int64(
					maximum,
				),
			),
		)

	if err != nil {
		return 0, err
	}

	return int(
		draw.Int64(),
	), nil
}

func clampPVEChance(
	value int,
	minimum int,
	maximum int,
) int {
	if value < minimum {
		return minimum
	}

	if value > maximum {
		return maximum
	}

	return value
}
