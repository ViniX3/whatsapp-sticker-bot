package rpg

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

var ErrInvalidRaidBossCatalog = errors.New(
	"catálogo de Raid Boss inválido",
)

var ErrRaidBossNotFound = errors.New(
	"Raid Boss não encontrado",
)

var (
	raidBossCatalogOnce sync.Once

	cachedRaidBossCatalog *RaidBossCatalog

	cachedRaidBossCatalogErr error
)

//go:embed data/raid_bosses.json
var raidBossData []byte

type RaidBoss struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Rarity Rarity `json:"rarity"`

	Power int `json:"power"`

	SetID string `json:"set_id"`

	ImageKey string `json:"image_key"`

	Description string `json:"description"`
}

type RaidBossCatalog struct {
	bosses map[string]RaidBoss

	ordered []RaidBoss
}

func LoadRaidBossCatalog() (
	*RaidBossCatalog,
	error,
) {

	var bosses []RaidBoss

	if err :=
		json.Unmarshal(
			raidBossData,
			&bosses,
		); err != nil {

		return nil,
			fmt.Errorf(
				"%w: erro lendo JSON: %v",
				ErrInvalidRaidBossCatalog,
				err,
			)
	}

	if len(bosses) == 0 {
		return nil,
			fmt.Errorf(
				"%w: catálogo vazio",
				ErrInvalidRaidBossCatalog,
			)
	}

	catalog :=
		&RaidBossCatalog{
			bosses: make(
				map[string]RaidBoss,
				len(bosses),
			),

			ordered: make(
				[]RaidBoss,
				0,
				len(bosses),
			),
		}

	imageKeys :=
		make(
			map[string]string,
			len(bosses),
		)

	setIDs :=
		make(
			map[string]string,
			len(bosses),
		)

	for _, boss := range bosses {

		boss.ID =
			strings.ToLower(
				strings.TrimSpace(
					boss.ID,
				),
			)

		boss.Name =
			strings.TrimSpace(
				boss.Name,
			)

		boss.SetID =
			strings.ToLower(
				strings.TrimSpace(
					boss.SetID,
				),
			)

		boss.ImageKey =
			strings.ToLower(
				strings.TrimSpace(
					boss.ImageKey,
				),
			)

		boss.Description =
			strings.TrimSpace(
				boss.Description,
			)

		if err :=
			validateRaidBoss(
				boss,
			); err != nil {

			return nil, err
		}

		if _, exists :=
			catalog.bosses[boss.ID]; exists {

			return nil,
				fmt.Errorf(
					"%w: ID duplicado %q",
					ErrInvalidRaidBossCatalog,
					boss.ID,
				)
		}

		if otherBoss,
			exists :=
			imageKeys[boss.ImageKey]; exists {

			return nil,
				fmt.Errorf(
					"%w: image_key %q usado por %q e %q",
					ErrInvalidRaidBossCatalog,
					boss.ImageKey,
					otherBoss,
					boss.ID,
				)
		}

		if otherBoss,
			exists :=
			setIDs[boss.SetID]; exists {

			return nil,
				fmt.Errorf(
					"%w: set_id %q usado por %q e %q",
					ErrInvalidRaidBossCatalog,
					boss.SetID,
					otherBoss,
					boss.ID,
				)
		}

		catalog.bosses[boss.ID] = boss

		imageKeys[boss.ImageKey] = boss.ID

		setIDs[boss.SetID] = boss.ID

		catalog.ordered =
			append(
				catalog.ordered,
				boss,
			)
	}

	sort.SliceStable(
		catalog.ordered,
		func(
			i int,
			j int,
		) bool {

			left :=
				catalog.ordered[i]

			right :=
				catalog.ordered[j]

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

	return catalog,
		nil
}

func validateRaidBoss(
	boss RaidBoss,
) error {

	if boss.ID == "" {
		return fmt.Errorf(
			"%w: Boss sem ID",
			ErrInvalidRaidBossCatalog,
		)
	}

	if strings.ContainsAny(
		boss.ID,
		" \t\r\n",
	) {

		return fmt.Errorf(
			"%w: ID inválido %q",
			ErrInvalidRaidBossCatalog,
			boss.ID,
		)
	}

	if boss.Name == "" {
		return fmt.Errorf(
			"%w: Boss %q sem nome",
			ErrInvalidRaidBossCatalog,
			boss.ID,
		)
	}

	switch boss.Rarity {

	case RarityLegendary,
		RarityMythic,
		RaritySacred:

	default:

		return fmt.Errorf(
			"%w: Boss %q possui raridade %q",
			ErrInvalidRaidBossCatalog,
			boss.ID,
			boss.Rarity,
		)
	}

	if boss.Power <= 0 {
		return fmt.Errorf(
			"%w: Boss %q possui poder %d",
			ErrInvalidRaidBossCatalog,
			boss.ID,
			boss.Power,
		)
	}

	if boss.SetID == "" {
		return fmt.Errorf(
			"%w: Boss %q sem set_id",
			ErrInvalidRaidBossCatalog,
			boss.ID,
		)
	}

	expectedPrefix :=
		strings.ToLower(
			string(
				boss.Rarity,
			),
		) + "_"

	if !strings.HasPrefix(
		boss.SetID,
		expectedPrefix,
	) {

		return fmt.Errorf(
			"%w: Boss %q possui set_id %q incompatível com %s",
			ErrInvalidRaidBossCatalog,
			boss.ID,
			boss.SetID,
			boss.Rarity,
		)
	}

	if boss.ImageKey == "" {
		return fmt.Errorf(
			"%w: Boss %q sem image_key",
			ErrInvalidRaidBossCatalog,
			boss.ID,
		)
	}

	// Mantemos o nome do asset diretamente ligado
	// ao ID do Boss.
	if boss.ImageKey !=
		boss.ID {

		return fmt.Errorf(
			"%w: Boss %q possui image_key %q",
			ErrInvalidRaidBossCatalog,
			boss.ID,
			boss.ImageKey,
		)
	}

	if boss.Description == "" {
		return fmt.Errorf(
			"%w: Boss %q sem descrição",
			ErrInvalidRaidBossCatalog,
			boss.ID,
		)
	}

	return nil
}

func (
	catalog *RaidBossCatalog,
) BossByID(
	bossID string,
) (
	RaidBoss,
	bool,
) {

	if catalog == nil {
		return RaidBoss{},
			false
	}

	bossID =
		strings.ToLower(
			strings.TrimSpace(
				bossID,
			),
		)

	boss, exists :=
		catalog.bosses[bossID]

	return boss,
		exists
}

func (
	catalog *RaidBossCatalog,
) Bosses() []RaidBoss {

	if catalog == nil {
		return nil
	}

	result :=
		make(
			[]RaidBoss,
			len(
				catalog.ordered,
			),
		)

	copy(
		result,
		catalog.ordered,
	)

	return result
}

func (
	catalog *RaidBossCatalog,
) BossesByRarity(
	rarity Rarity,
) []RaidBoss {

	if catalog == nil {
		return nil
	}

	result :=
		make(
			[]RaidBoss,
			0,
		)

	for _, boss := range catalog.ordered {

		if boss.Rarity !=
			rarity {

			continue
		}

		result =
			append(
				result,
				boss,
			)
	}

	return result
}

// defaultRaidBossCatalog mantém uma única instância
// validada do catálogo durante a execução do bot.
func defaultRaidBossCatalog() (
	*RaidBossCatalog,
	error,
) {

	raidBossCatalogOnce.Do(
		func() {

			cachedRaidBossCatalog,
				cachedRaidBossCatalogErr =
				LoadRaidBossCatalog()
		},
	)

	return cachedRaidBossCatalog,
		cachedRaidBossCatalogErr
}

// RaidBosses retorna todos os Raid Bosses
// usando o catálogo padrão cacheado.
func RaidBosses() []RaidBoss {

	catalog, err :=
		defaultRaidBossCatalog()

	if err != nil ||
		catalog == nil {

		return nil
	}

	return catalog.Bosses()
}

// RaidBossByID procura um Raid Boss pelo ID.
func RaidBossByID(
	bossID string,
) (
	RaidBoss,
	bool,
) {

	catalog, err :=
		defaultRaidBossCatalog()

	if err != nil ||
		catalog == nil {

		return RaidBoss{},
			false
	}

	return catalog.BossByID(
		bossID,
	)
}

// RaidBossesByRarity retorna apenas os Bosses
// da raridade informada.
func RaidBossesByRarity(
	rarity Rarity,
) []RaidBoss {

	catalog, err :=
		defaultRaidBossCatalog()

	if err != nil ||
		catalog == nil {

		return nil
	}

	return catalog.BossesByRarity(
		rarity,
	)
}
