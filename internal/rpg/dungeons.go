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

var (
	ErrDungeonNotFound = errors.New(
		"dungeon não encontrada",
	)

	ErrDungeonLootUnavailable = errors.New(
		"nenhum loot disponível para a dungeon",
	)
)

type DungeonCooldownError struct {
	Remaining time.Duration
}

func (err *DungeonCooldownError) Error() string {
	return "dungeon em cooldown"
}

type Dungeon struct {
	ID          string
	Name        string
	Emoji       string
	Description string

	RecommendedPower int

	GoldMin int
	GoldMax int

	CrystalMin int
	CrystalMax int

	MinLootRarity Rarity
	MaxLootRarity Rarity

	Cooldown time.Duration
}

type DungeonAttemptResult struct {
	Dungeon Dungeon

	Won bool

	PlayerPower int

	SuccessChance int
	SuccessBonus  int

	GoldReward int
	Balance    int

	CrystalReward  int
	CrystalBalance int

	Loot         *Material
	LootQuantity int
}

var dungeonCatalog = []Dungeon{
	{
		ID: "ruinas",

		Name: "Ruínas Abandonadas",

		Emoji: "🏚️",

		Description: "Antigas ruínas tomadas por criaturas e saqueadores.",

		RecommendedPower: 250,

		GoldMin: 100000,
		GoldMax: 150000,

		CrystalMin: 10,
		CrystalMax: 20,

		MinLootRarity: RarityCommon,
		MaxLootRarity: RarityRare,

		Cooldown: time.Minute,
	},
	{
		ID: "cripta",

		Name: "Cripta Sombria",

		Emoji: "⚰️",

		Description: "Uma cripta esquecida onde forças sombrias guardam tesouros antigos.",

		RecommendedPower: 600,

		GoldMin: 300000,
		GoldMax: 500000,

		CrystalMin: 25,
		CrystalMax: 50,

		MinLootRarity: RarityRare,
		MaxLootRarity: RarityEpic,

		Cooldown: time.Minute,
	},
	{
		ID: "fortaleza",

		Name: "Fortaleza Amaldiçoada",

		Emoji: "🏰",

		Description: "Uma fortaleza dominada por guerreiros e criaturas de grande poder.",

		RecommendedPower: 1200,

		GoldMin: 800000,
		GoldMax: 1200000,

		CrystalMin: 60,
		CrystalMax: 100,

		MinLootRarity: RarityEpic,
		MaxLootRarity: RarityLegendary,

		Cooldown: time.Minute,
	},
}

func Dungeons() []Dungeon {
	result :=
		make(
			[]Dungeon,
			len(dungeonCatalog),
		)

	copy(
		result,
		dungeonCatalog,
	)

	return result
}

func DungeonByID(
	value string,
) (Dungeon, bool) {

	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	switch value {

	case "1",
		"ruina",
		"ruinas",
		"ruína",
		"ruínas":

		value = "ruinas"

	case "2",
		"cripta":

		value = "cripta"

	case "3",
		"fortaleza":

		value = "fortaleza"
	}

	for _, dungeon := range dungeonCatalog {

		if dungeon.ID == value {
			return dungeon,
				true
		}
	}

	return Dungeon{},
		false
}

func DungeonSuccessBonus(
	summary *EquipmentSummary,
) int {

	if summary == nil {
		return 0
	}

	bonus := 0

	for _, active := range summary.ActiveSetBonuses {

		for _, effect := range active.Effects {

			if effect.Type !=
				BonusDungeonSuccessPercent {

				continue
			}

			bonus +=
				effect.Value
		}
	}

	return bonus
}

func DungeonCrystalRewardBonus(
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

func applyDungeonCrystalBonus(
	amount int,
	bonusPercent int,
) int {

	if amount <= 0 {
		return 0
	}

	if bonusPercent <= 0 {
		return amount
	}

	bonus :=
		amount *
			bonusPercent /
			100

	if bonus < 1 {
		bonus = 1
	}

	return amount +
		bonus
}

func DungeonSuccessChance(
	playerPower int,
	recommendedPower int,
	bonus int,
) int {

	if playerPower <= 0 {
		return 0
	}

	if recommendedPower <= 0 {
		return 100
	}

	// Jogadores com pelo menos o dobro do PC recomendado
	// já dominam completamente esta dungeon.
	if playerPower >=
		recommendedPower*2 {

		return 100
	}

	baseChance := 5

	// Muito abaixo do poder recomendado.
	if playerPower*2 <
		recommendedPower {

		baseChance = 5

	} else if playerPower >=
		recommendedPower*3 {

		// Jogador extremamente superior.
		baseChance = 98

	} else {

		// Reutiliza a curva PvE balanceada,
		// mas dungeons recebem +15 pontos,
		// pois o PC indicado representa o
		// poder recomendado, não uma luta 50/50.
		baseChance =
			PVEWinChance(
				playerPower,
				recommendedPower,
				false,
			) + 15

		if baseChance > 95 {
			baseChance = 95
		}
	}

	chance :=
		baseChance +
			bonus

	if chance < 5 {
		chance = 5
	}

	if chance > 98 {
		chance = 98
	}

	return chance
}

func AttemptDungeon(
	groupJID string,
	jid string,
	dungeonID string,
	catalog *Catalog,
	materials *MaterialCatalog,
) (*DungeonAttemptResult, error) {

	dungeon, exists :=
		DungeonByID(
			dungeonID,
		)

	if !exists {
		return nil,
			ErrDungeonNotFound
	}

	if catalog == nil {
		return nil,
			errors.New(
				"catálogo RPG não inicializado",
			)
	}

	if materials == nil {
		return nil,
			errors.New(
				"catálogo de materiais não inicializado",
			)
	}

	if _, err :=
		GetOrCreatePlayer(
			groupJID,
			jid,
		); err != nil {

		return nil, err
	}

	summary, err :=
		GetEquipmentSummary(
			groupJID,
			jid,
			catalog,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando equipamentos da dungeon: %w",
				err,
			)
	}

	bonus :=
		DungeonSuccessBonus(
			summary,
		)

	chance :=
		DungeonSuccessChance(
			summary.CombatPower,
			dungeon.RecommendedPower,
			bonus,
		)

	if err :=
		ensureDungeonSchema(); err != nil {

		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando dungeon: %w",
				err,
			)
	}

	defer tx.Rollback()

	now :=
		time.Now()

	var lastAttemptUnix int64

	err =
		tx.QueryRow(`
			SELECT last_attempt_unix
			FROM rpg_dungeon_progress
			WHERE group_jid = ?
			  AND jid = ?
			  AND dungeon_id = ?
		`,
			groupJID,
			jid,
			dungeon.ID,
		).Scan(
			&lastAttemptUnix,
		)

	if err != nil &&
		!errors.Is(
			err,
			sql.ErrNoRows,
		) {

		return nil,
			fmt.Errorf(
				"erro consultando cooldown da dungeon: %w",
				err,
			)
	}

	if err == nil &&
		lastAttemptUnix > 0 {

		nextAttempt :=
			time.Unix(
				lastAttemptUnix,
				0,
			).Add(
				dungeon.Cooldown,
			)

		if now.Before(
			nextAttempt,
		) {

			return nil,
				&DungeonCooldownError{
					Remaining: time.Until(
						nextAttempt,
					),
				}
		}
	}

	won, err :=
		dungeonRoll(
			chance,
		)

	if err != nil {
		return nil, err
	}

	result :=
		&DungeonAttemptResult{
			Dungeon: dungeon,

			Won: won,

			PlayerPower: summary.CombatPower,

			SuccessChance: chance,

			SuccessBonus: bonus,
		}

	if won {

		goldReward, err :=
			dungeonRandomRange(
				dungeon.GoldMin,
				dungeon.GoldMax,
			)

		if err != nil {
			return nil, err
		}

		crystalReward, err :=
			dungeonRandomRange(
				dungeon.CrystalMin,
				dungeon.CrystalMax,
			)

		if err != nil {
			return nil, err
		}

		crystalReward =
			applyDungeonCrystalBonus(
				crystalReward,
				DungeonCrystalRewardBonus(
					summary,
				),
			)

		loot,
			quantity,
			err :=
			rollDungeonLoot(
				dungeon,
				materials,
			)

		if err != nil {
			return nil, err
		}

		updateResult, err :=
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
			return nil,
				fmt.Errorf(
					"erro creditando Gold da dungeon: %w",
					err,
				)
		}

		rows, err :=
			updateResult.RowsAffected()

		if err != nil {
			return nil, err
		}

		if rows == 0 {
			return nil,
				ErrWalletNotFound
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
				crystalReward,
				groupJID,
				jid,
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro creditando cristais da dungeon: %w",
					err,
				)
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
				VALUES (
					?,
					?,
					NULL,
					?,
					'DUNGEON_REWARD',
					?
				)
			`,
				groupJID,
				jid,
				goldReward,
				fmt.Sprintf(
					"Recompensa da dungeon %s",
					dungeon.Name,
				),
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro registrando Gold da dungeon: %w",
					err,
				)
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
				loot.ID,
				quantity,
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro adicionando loot da dungeon: %w",
					err,
				)
		}

		result.GoldReward =
			goldReward

		result.CrystalReward =
			crystalReward

		result.Loot =
			&loot

		result.LootQuantity =
			quantity
	}

	winValue := 0

	if won {
		winValue = 1
	}

	_, err =
		tx.Exec(`
			INSERT INTO rpg_dungeon_progress (
				group_jid,
				jid,
				dungeon_id,
				attempts,
				wins,
				last_attempt_unix
			)
			VALUES (?, ?, ?, 1, ?, ?)

			ON CONFLICT (
				group_jid,
				jid,
				dungeon_id
			)
			DO UPDATE SET
				attempts =
					rpg_dungeon_progress.attempts + 1,

				wins =
					rpg_dungeon_progress.wins
					+ excluded.wins,

				last_attempt_unix =
					excluded.last_attempt_unix
		`,
			groupJID,
			jid,
			dungeon.ID,
			winValue,
			now.Unix(),
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro registrando progresso da dungeon: %w",
				err,
			)
	}

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
			&result.Balance,
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
				"erro consultando saldo após dungeon: %w",
				err,
			)
	}

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
			&result.CrystalBalance,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando cristais após dungeon: %w",
				err,
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando dungeon: %w",
				err,
			)
	}

	return result,
		nil
}

func ensureDungeonSchema() error {

	if err :=
		ensureSchema(); err != nil {

		return err
	}

	_, err :=
		database.DB.Exec(`
			CREATE TABLE IF NOT EXISTS
				rpg_dungeon_progress (

				group_jid TEXT NOT NULL,

				jid TEXT NOT NULL,

				dungeon_id TEXT NOT NULL,

				attempts INTEGER NOT NULL
					DEFAULT 0
					CHECK (attempts >= 0),

				wins INTEGER NOT NULL
					DEFAULT 0
					CHECK (wins >= 0),

				last_attempt_unix INTEGER NOT NULL
					DEFAULT 0,

				PRIMARY KEY (
					group_jid,
					jid,
					dungeon_id
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
			"erro criando schema de dungeons: %w",
			err,
		)
	}

	return nil
}

func rollDungeonLoot(
	dungeon Dungeon,
	materials *MaterialCatalog,
) (
	Material,
	int,
	error,
) {

	pool :=
		make(
			[]Material,
			0,
		)

	for _, material := range materials.Materials() {

		// Nesta primeira versão usamos somente
		// materiais genéricos de coleta.
		//
		// Isso impede que uma dungeon entregue
		// diretamente partes exclusivas de Boss
		// ou do Mercador de Outro Mundo.
		if material.Source !=
			MaterialSourceGathering {

			continue
		}

		rank :=
			material.Rarity.Rank()

		if rank <
			dungeon.MinLootRarity.Rank() ||
			rank >
				dungeon.MaxLootRarity.Rank() {

			continue
		}

		pool =
			append(
				pool,
				material,
			)
	}

	if len(pool) == 0 {
		return Material{},
			0,
			ErrDungeonLootUnavailable
	}

	index, err :=
		dungeonRandomInt(
			len(pool),
		)

	if err != nil {
		return Material{},
			0,
			err
	}

	material :=
		pool[index]

	quantity := 1

	switch material.Rarity {

	case RarityCommon:

		quantity, err =
			dungeonRandomRange(
				2,
				4,
			)

	case RarityRare:

		quantity, err =
			dungeonRandomRange(
				1,
				3,
			)

	case RarityEpic:

		quantity, err =
			dungeonRandomRange(
				1,
				2,
			)

	case RarityLegendary:

		quantity = 1
	}

	if err != nil {
		return Material{},
			0,
			err
	}

	return material,
		quantity,
		nil
}

func dungeonRoll(
	chance int,
) (bool, error) {

	if chance <= 0 {
		return false, nil
	}

	if chance >= 100 {
		return true, nil
	}

	value, err :=
		dungeonRandomInt(
			100,
		)

	if err != nil {
		return false, err
	}

	return value < chance,
		nil
}

func dungeonRandomRange(
	minimum int,
	maximum int,
) (int, error) {

	if minimum > maximum {
		return 0,
			fmt.Errorf(
				"intervalo inválido: %d-%d",
				minimum,
				maximum,
			)
	}

	size :=
		maximum -
			minimum +
			1

	value, err :=
		dungeonRandomInt(
			size,
		)

	if err != nil {
		return 0, err
	}

	return minimum +
			value,
		nil
}

func dungeonRandomInt(
	maximum int,
) (int, error) {

	if maximum <= 0 {
		return 0,
			fmt.Errorf(
				"limite aleatório inválido: %d",
				maximum,
			)
	}

	value, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(
				int64(maximum),
			),
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro gerando sorteio da dungeon: %w",
				err,
			)
	}

	return int(
			value.Int64(),
		),
		nil
}
