package rpg

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	EnemyCommonMinPower = 550
	EnemyCommonMaxPower = 750

	EnemyRareMinPower = 1050
	EnemyRareMaxPower = 1450

	EnemyEpicMinPower = 1900
	EnemyEpicMaxPower = 2500

	EnemyBossMinPower = 2900
	EnemyBossMaxPower = 3400
)

type BossBlessing struct {
	ID string `json:"id"`

	Name string `json:"name"`

	DurationSeconds int `json:"duration_seconds"`

	GlobalExtraGatherRolls int `json:"global_extra_gather_rolls"`

	RegionQuantityBonusPercent int `json:"region_quantity_bonus_percent"`

	RegionRarityUpgradeChancePercent int `json:"region_rarity_upgrade_chance_percent"`
}

type Enemy struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Region GatheringRegion `json:"region"`

	Rarity Rarity `json:"rarity"`

	Power int `json:"power"`

	Boss bool `json:"boss"`

	GoldMin int `json:"gold_min"`

	GoldMax int `json:"gold_max"`

	CrystalDropChancePercent int `json:"crystal_drop_chance_percent"`

	CrystalMin int `json:"crystal_min"`

	CrystalMax int `json:"crystal_max"`

	Description string `json:"description"`

	Blessing *BossBlessing `json:"blessing,omitempty"`
}

//go:embed data/enemies.json
var enemyFS embed.FS

type EnemyCatalog struct {
	enemies map[string]Enemy

	ordered []Enemy
}

func LoadEnemyCatalog() (
	*EnemyCatalog,
	error,
) {
	data, err :=
		enemyFS.ReadFile(
			"data/enemies.json",
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro lendo catálogo de monstros: %w",
				err,
			)
	}

	var enemies []Enemy

	if err :=
		json.Unmarshal(
			data,
			&enemies,
		); err != nil {

		return nil,
			fmt.Errorf(
				"erro decodificando catálogo de monstros: %w",
				err,
			)
	}

	catalog :=
		&EnemyCatalog{
			enemies: make(
				map[string]Enemy,
			),

			ordered: make(
				[]Enemy,
				0,
				len(enemies),
			),
		}

	for _, enemy := range enemies {
		if err :=
			validateEnemy(
				enemy,
			); err != nil {

			return nil,
				fmt.Errorf(
					"monstro inválido %q: %w",
					enemy.ID,
					err,
				)
		}

		if _, exists :=
			catalog.enemies[enemy.ID]; exists {

			return nil,
				fmt.Errorf(
					"id de monstro duplicado: %s",
					enemy.ID,
				)
		}

		catalog.enemies[enemy.ID] = enemy

		catalog.ordered =
			append(
				catalog.ordered,
				enemy,
			)
	}

	if err :=
		validateEnemyDistribution(
			catalog.ordered,
		); err != nil {

		return nil, err
	}

	sort.Slice(
		catalog.ordered,
		func(
			i int,
			j int,
		) bool {
			left :=
				catalog.ordered[i]

			right :=
				catalog.ordered[j]

			if left.Region !=
				right.Region {

				return left.Region <
					right.Region
			}

			if left.Rarity.Rank() !=
				right.Rarity.Rank() {

				return left.Rarity.Rank() <
					right.Rarity.Rank()
			}

			if left.Power !=
				right.Power {

				return left.Power <
					right.Power
			}

			return left.ID <
				right.ID
		},
	)

	return catalog, nil
}

func validateEnemy(
	enemy Enemy,
) error {
	if strings.TrimSpace(
		enemy.ID,
	) == "" {

		return fmt.Errorf(
			"id vazio",
		)
	}

	if strings.TrimSpace(
		enemy.Name,
	) == "" {

		return fmt.Errorf(
			"nome vazio",
		)
	}

	if !enemy.Region.Valid() {
		return fmt.Errorf(
			"região inválida: %s",
			enemy.Region,
		)
	}

	switch enemy.Rarity {
	case RarityCommon,
		RarityRare,
		RarityEpic,
		RarityLegendary:

	default:
		return fmt.Errorf(
			"raridade inválida para PvE: %s",
			enemy.Rarity,
		)
	}

	if enemy.Power <= 0 {
		return fmt.Errorf(
			"PC inválido: %d",
			enemy.Power,
		)
	}

	minPower,
		maxPower :=
		enemyPowerRange(
			enemy.Rarity,
		)

	if enemy.Power < minPower ||
		enemy.Power > maxPower {

		return fmt.Errorf(
			"PC %d fora da faixa %d-%d para %s",
			enemy.Power,
			minPower,
			maxPower,
			enemy.Rarity,
		)
	}

	if enemy.GoldMin < 0 ||
		enemy.GoldMax <
			enemy.GoldMin {

		return fmt.Errorf(
			"faixa de Gold inválida: %d-%d",
			enemy.GoldMin,
			enemy.GoldMax,
		)
	}

	if enemy.CrystalDropChancePercent < 0 ||
		enemy.CrystalDropChancePercent > 100 {

		return fmt.Errorf(
			"chance de Cristal inválida: %d",
			enemy.CrystalDropChancePercent,
		)
	}

	if enemy.CrystalMin < 0 ||
		enemy.CrystalMax <
			enemy.CrystalMin {

		return fmt.Errorf(
			"faixa de Cristal inválida: %d-%d",
			enemy.CrystalMin,
			enemy.CrystalMax,
		)
	}

	if enemy.Boss {
		if enemy.Rarity !=
			RarityLegendary {

			return fmt.Errorf(
				"boss precisa ser LEGENDARY",
			)
		}

		if enemy.Blessing == nil {
			return fmt.Errorf(
				"boss não possui bênção",
			)
		}

		if err :=
			validateBossBlessing(
				*enemy.Blessing,
			); err != nil {

			return err
		}

	} else {
		if enemy.Rarity ==
			RarityLegendary {

			return fmt.Errorf(
				"LEGENDARY regional precisa ser boss",
			)
		}

		if enemy.Blessing != nil {
			return fmt.Errorf(
				"monstro comum não pode possuir bênção de boss",
			)
		}
	}

	return nil
}

func validateBossBlessing(
	blessing BossBlessing,
) error {
	if strings.TrimSpace(
		blessing.ID,
	) == "" {

		return fmt.Errorf(
			"id da bênção vazio",
		)
	}

	if strings.TrimSpace(
		blessing.Name,
	) == "" {

		return fmt.Errorf(
			"nome da bênção vazio",
		)
	}

	if blessing.DurationSeconds <= 0 {
		return fmt.Errorf(
			"duração da bênção inválida",
		)
	}

	if blessing.GlobalExtraGatherRolls < 0 {
		return fmt.Errorf(
			"rolagens extras inválidas",
		)
	}

	if blessing.RegionQuantityBonusPercent < 0 ||
		blessing.RegionQuantityBonusPercent > 100 {

		return fmt.Errorf(
			"bônus de quantidade inválido",
		)
	}

	if blessing.RegionRarityUpgradeChancePercent < 0 ||
		blessing.RegionRarityUpgradeChancePercent > 100 {

		return fmt.Errorf(
			"bônus de raridade inválido",
		)
	}

	return nil
}

func enemyPowerRange(
	rarity Rarity,
) (
	int,
	int,
) {
	switch rarity {
	case RarityCommon:
		return EnemyCommonMinPower,
			EnemyCommonMaxPower

	case RarityRare:
		return EnemyRareMinPower,
			EnemyRareMaxPower

	case RarityEpic:
		return EnemyEpicMinPower,
			EnemyEpicMaxPower

	case RarityLegendary:
		return EnemyBossMinPower,
			EnemyBossMaxPower
	}

	return 0, 0
}

func validateEnemyDistribution(
	enemies []Enemy,
) error {
	expected :=
		map[Rarity]int{
			RarityCommon:    15,
			RarityRare:      10,
			RarityEpic:      5,
			RarityLegendary: 1,
		}

	regions :=
		[]GatheringRegion{
			GatheringForest,
			GatheringQuarry,
			GatheringMine,
		}

	if len(enemies) != 93 {
		return fmt.Errorf(
			"catálogo PvE deve possuir 93 monstros; possui %d",
			len(enemies),
		)
	}

	blessingIDs :=
		make(
			map[string]struct{},
		)

	for _, region := range regions {

		counts :=
			make(
				map[Rarity]int,
			)

		bosses := 0

		for _, enemy := range enemies {

			if enemy.Region !=
				region {

				continue
			}

			counts[enemy.Rarity]++

			if enemy.Boss {
				bosses++

				blessingID :=
					enemy.Blessing.ID

				if _, exists :=
					blessingIDs[blessingID]; exists {

					return fmt.Errorf(
						"bênção duplicada: %s",
						blessingID,
					)
				}

				blessingIDs[blessingID] = struct{}{}
			}
		}

		for rarity, expectedCount := range expected {

			if counts[rarity] !=
				expectedCount {

				return fmt.Errorf(
					"região %s possui %d monstros %s; esperado %d",
					region,
					counts[rarity],
					rarity,
					expectedCount,
				)
			}
		}

		if bosses != 1 {
			return fmt.Errorf(
				"região %s precisa possuir exatamente 1 boss; possui %d",
				region,
				bosses,
			)
		}
	}

	return nil
}

func (
	catalog *EnemyCatalog,
) EnemyByID(
	enemyID string,
) (
	Enemy,
	bool,
) {
	enemy, exists :=
		catalog.enemies[enemyID]

	return enemy, exists
}

func (
	catalog *EnemyCatalog,
) Enemies() []Enemy {
	result :=
		make(
			[]Enemy,
			len(catalog.ordered),
		)

	copy(
		result,
		catalog.ordered,
	)

	return result
}

func (
	catalog *EnemyCatalog,
) EnemiesByRegion(
	region GatheringRegion,
) []Enemy {
	result :=
		make(
			[]Enemy,
			0,
		)

	for _, enemy := range catalog.ordered {

		if enemy.Region ==
			region {

			result =
				append(
					result,
					enemy,
				)
		}
	}

	return result
}

func (
	catalog *EnemyCatalog,
) EnemiesByRegionAndRarity(
	region GatheringRegion,
	rarity Rarity,
) []Enemy {
	result :=
		make(
			[]Enemy,
			0,
		)

	for _, enemy := range catalog.ordered {

		if enemy.Region !=
			region ||
			enemy.Rarity !=
				rarity {

			continue
		}

		result =
			append(
				result,
				enemy,
			)
	}

	return result
}

func (
	catalog *EnemyCatalog,
) BossByRegion(
	region GatheringRegion,
) (
	Enemy,
	bool,
) {
	for _, enemy := range catalog.ordered {

		if enemy.Region ==
			region &&
			enemy.Boss {

			return enemy, true
		}
	}

	return Enemy{}, false
}
