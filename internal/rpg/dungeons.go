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

	ErrDungeonLocked = errors.New(
		"dungeon selada",
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

	EquipmentDropChance int
	EquipmentPrefixes   []string

	Chaos  bool
	Locked bool

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

	EquipmentDrop *Item
}

var dungeonCatalog = []Dungeon{
	{
		ID:                  "ruinas",
		Name:                "Ruínas Abandonadas",
		Emoji:               "🏚️",
		Description:         "O primeiro desafio após o fim da progressão PvE.",
		RecommendedPower:    8000,
		GoldMin:             800000,
		GoldMax:             1200000,
		CrystalMin:          100,
		CrystalMax:          200,
		MinLootRarity:       RarityRare,
		MaxLootRarity:       RarityEpic,
		EquipmentDropChance: 10,
		EquipmentPrefixes: []string{
			"legendary_blood_moon_wolf",
			"legendary_storm_griffin",
		},
		Cooldown: time.Minute,
	},
	{
		ID:                  "cripta",
		Name:                "Cripta Sombria",
		Emoji:               "⚰️",
		Description:         "Uma necrópole profunda onde os primeiros Lendários aguardam.",
		RecommendedPower:    9500,
		GoldMin:             1500000,
		GoldMax:             2200000,
		CrystalMin:          200,
		CrystalMax:          350,
		MinLootRarity:       RarityRare,
		MaxLootRarity:       RarityEpic,
		EquipmentDropChance: 12,
		EquipmentPrefixes: []string{
			"legendary_leviathan",
			"legendary_frost_wyrm",
		},
		Cooldown: time.Minute,
	},
	{
		ID:                  "fortaleza",
		Name:                "Fortaleza Amaldiçoada",
		Emoji:               "🏰",
		Description:         "Uma cidadela em guerra permanente, guardada por seres ancestrais.",
		RecommendedPower:    11000,
		GoldMin:             2500000,
		GoldMax:             3500000,
		CrystalMin:          350,
		CrystalMax:          550,
		MinLootRarity:       RarityEpic,
		MaxLootRarity:       RarityEpic,
		EquipmentDropChance: 14,
		EquipmentPrefixes: []string{
			"legendary_black_dragon",
			"legendary_ancient_demon",
		},
		Cooldown: time.Minute,
	},
	{
		ID:                  "templo",
		Name:                "Templo do Abismo",
		Emoji:               "🕯️",
		Description:         "Um templo soterrado onde antigas entidades ainda recebem oferendas.",
		RecommendedPower:    13000,
		GoldMin:             4000000,
		GoldMax:             6000000,
		CrystalMin:          550,
		CrystalMax:          800,
		MinLootRarity:       RarityEpic,
		MaxLootRarity:       RarityEpic,
		EquipmentDropChance: 17,
		EquipmentPrefixes: []string{
			"legendary_behemoth",
			"legendary_infernal_colossus",
		},
		Cooldown: time.Minute,
	},
	{
		ID:                  "labirinto",
		Name:                "Labirinto do Rei Caído",
		Emoji:               "👑",
		Description:         "Corredores sem fim escondem criaturas que destruíram antigos reinos.",
		RecommendedPower:    15500,
		GoldMin:             6500000,
		GoldMax:             9000000,
		CrystalMin:          800,
		CrystalMax:          1200,
		MinLootRarity:       RarityEpic,
		MaxLootRarity:       RarityLegendary,
		EquipmentDropChance: 20,
		EquipmentPrefixes: []string{
			"legendary_obsidian_basilisk",
		},
		Cooldown: time.Minute,
	},
	{
		ID:                  "trono",
		Name:                "Trono dos Antigos",
		Emoji:               "🗿",
		Description:         "O limite das Dungeons do mundo normal e a última prova antes do endgame.",
		RecommendedPower:    18000,
		GoldMin:             10000000,
		GoldMax:             15000000,
		CrystalMin:          1200,
		CrystalMax:          1800,
		MinLootRarity:       RarityEpic,
		MaxLootRarity:       RarityLegendary,
		EquipmentDropChance: 25,
		EquipmentPrefixes: []string{
			"legendary_abyssal_kraken",
		},
		Cooldown: time.Minute,
	},
	{
		ID:               "fenda-caos",
		Name:             "Fenda do Caos",
		Emoji:            "🌀",
		Description:      "Uma ruptura impossível na realidade. Algo observa do outro lado.",
		RecommendedPower: 100000,
		Chaos:            true,
		Locked:           true,
		Cooldown:         5 * time.Minute,
	},
	{
		ID:               "catedral-vazio",
		Name:             "Catedral do Vazio",
		Emoji:            "🌑",
		Description:      "Uma construção de outro mundo onde as leis naturais deixaram de existir.",
		RecommendedPower: 175000,
		Chaos:            true,
		Locked:           true,
		Cooldown:         5 * time.Minute,
	},
	{
		ID:               "coracao-fim",
		Name:             "Coração do Fim",
		Emoji:            "☠️",
		Description:      "O lugar onde até os monstros do Caos temem entrar.",
		RecommendedPower: 300000,
		Chaos:            true,
		Locked:           true,
		Cooldown:         10 * time.Minute,
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

func NormalDungeons() []Dungeon {
	result := make([]Dungeon, 0, 6)

	for _, dungeon := range dungeonCatalog {
		if dungeon.Chaos {
			continue
		}
		result = append(result, dungeon)
	}

	return result
}

func ChaosDungeons() []Dungeon {
	result := make([]Dungeon, 0, 3)

	for _, dungeon := range dungeonCatalog {
		if !dungeon.Chaos {
			continue
		}
		result = append(result, dungeon)
	}

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
	case "1", "ruina", "ruinas", "ruína", "ruínas":
		value = "ruinas"
	case "2", "cripta":
		value = "cripta"
	case "3", "fortaleza":
		value = "fortaleza"
	case "4", "templo", "abismo", "templo-abismo":
		value = "templo"
	case "5", "labirinto", "rei-caido", "rei-caído":
		value = "labirinto"
	case "6", "trono", "antigos", "trono-antigos":
		value = "trono"
	case "7", "fenda", "fenda-caos":
		value = "fenda-caos"
	case "8", "catedral", "vazio", "catedral-vazio":
		value = "catedral-vazio"
	case "9", "coracao", "coração", "fim", "coracao-fim", "coração-fim":
		value = "coracao-fim"
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

func DungeonSuccessChanceFor(
	dungeon Dungeon,
	playerPower int,
	bonus int,
) int {
	if !dungeon.Chaos {
		return DungeonSuccessChance(
			playerPower,
			dungeon.RecommendedPower,
			bonus,
		)
	}

	if playerPower <= 0 ||
		dungeon.RecommendedPower <= 0 {
		return 0
	}

	ratio := playerPower * 100 / dungeon.RecommendedPower

	baseChance := 1

	switch {
	case ratio < 50:
		baseChance = 1
	case ratio < 75:
		baseChance = 3
	case ratio < 100:
		baseChance = 8
	case ratio < 125:
		baseChance = 18
	case ratio < 150:
		baseChance = 28
	case ratio < 200:
		baseChance = 40
	case ratio < 250:
		baseChance = 55
	case ratio < 300:
		baseChance = 65
	case ratio < 400:
		baseChance = 75
	case ratio < 500:
		baseChance = 85
	default:
		baseChance = 92
	}

	chance := baseChance + bonus

	if chance < 1 {
		chance = 1
	}

	if chance > 95 {
		chance = 95
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

	if dungeon.Locked {
		return nil,
			ErrDungeonLocked
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
		DungeonSuccessChanceFor(
			dungeon,
			summary.CombatPower,
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

		equipmentDrop, err :=
			rollDungeonEquipment(
				dungeon,
				catalog,
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

		if equipmentDrop != nil {
			_, err =
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
				return nil,
					fmt.Errorf(
						"erro adicionando equipamento da dungeon: %w",
						err,
					)
			}
		}

		result.GoldReward =
			goldReward

		result.CrystalReward =
			crystalReward

		result.Loot =
			&loot

		result.LootQuantity =
			quantity

		result.EquipmentDrop =
			equipmentDrop
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

func rollDungeonEquipment(
	dungeon Dungeon,
	catalog *Catalog,
) (
	*Item,
	error,
) {
	if dungeon.EquipmentDropChance <= 0 ||
		len(dungeon.EquipmentPrefixes) == 0 {
		return nil, nil
	}

	wonRoll, err :=
		dungeonRoll(
			dungeon.EquipmentDropChance,
		)

	if err != nil {
		return nil, err
	}

	if !wonRoll {
		return nil, nil
	}

	pool := make([]Item, 0, 16)

	for _, item := range catalog.ItemsByRarity(
		RarityLegendary,
	) {

		if !dungeonEquipmentAllowed(
			dungeon,
			item.ID,
		) {
			continue
		}

		pool = append(pool, item)
	}

	if len(pool) == 0 {
		return nil, nil
	}

	index, err :=
		dungeonRandomInt(
			len(pool),
		)

	if err != nil {
		return nil, err
	}

	item := pool[index]

	return &item, nil
}

func dungeonEquipmentAllowed(
	dungeon Dungeon,
	itemID string,
) bool {
	for _, prefix := range dungeon.EquipmentPrefixes {

		if strings.HasPrefix(
			itemID,
			prefix+"_",
		) {
			return true
		}
	}

	return false
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
