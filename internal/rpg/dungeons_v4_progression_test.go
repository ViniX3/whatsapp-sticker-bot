package rpg

import (
	"strings"
	"testing"
)

func TestDungeonV4Progression(
	t *testing.T,
) {
	expected :=
		map[string]int{
			"ruinas":         8000,
			"cripta":         9500,
			"fortaleza":      11000,
			"templo":         13000,
			"labirinto":      15500,
			"trono":          18000,
			"fenda-caos":     100000,
			"catedral-vazio": 175000,
			"coracao-fim":    300000,
		}

	dungeons := Dungeons()

	if len(dungeons) != 9 {
		t.Fatalf(
			"esperado 9 dungeons, recebido %d",
			len(dungeons),
		)
	}

	for id, power := range expected {
		dungeon, ok := DungeonByID(id)

		if !ok {
			t.Fatalf("dungeon %q não encontrada", id)
		}

		if dungeon.RecommendedPower != power {
			t.Fatalf(
				"%s: esperado PC %d, recebido %d",
				id,
				power,
				dungeon.RecommendedPower,
			)
		}
	}
}

func TestDungeonV4NormalAndChaosSplit(
	t *testing.T,
) {
	if got := len(NormalDungeons()); got != 6 {
		t.Fatalf(
			"esperado 6 normais, recebido %d",
			got,
		)
	}

	chaos := ChaosDungeons()

	if len(chaos) != 3 {
		t.Fatalf(
			"esperado 3 do Caos, recebido %d",
			len(chaos),
		)
	}

	for _, dungeon := range chaos {
		if !dungeon.Chaos {
			t.Fatalf(
				"%s deveria ser Caos",
				dungeon.ID,
			)
		}

		if !dungeon.Locked {
			t.Fatalf(
				"%s deveria iniciar selada",
				dungeon.ID,
			)
		}
	}
}

func TestDungeonV4MarketLegendarySetsAreReserved(
	t *testing.T,
) {
	reserved :=
		[]string{
			"legendary_crystal_golem_weapon",
			"legendary_onyx_spider_queen_weapon",
			"legendary_eternal_labyrinth_minotaur_weapon",
			"legendary_jade_dragon_weapon",
		}

	for _, dungeon := range NormalDungeons() {
		for _, itemID := range reserved {
			if dungeonEquipmentAllowed(
				dungeon,
				itemID,
			) {
				t.Fatalf(
					"%s não deveria dropar %s",
					dungeon.ID,
					itemID,
				)
			}
		}
	}
}

func TestChaosDungeonChanceNeverGuaranteed(
	t *testing.T,
) {
	dungeon, ok :=
		DungeonByID(
			"fenda-caos",
		)

	if !ok {
		t.Fatal("fenda-caos não encontrada")
	}

	chance :=
		DungeonSuccessChanceFor(
			dungeon,
			dungeon.RecommendedPower*10,
			100,
		)

	if chance > 95 {
		t.Fatalf(
			"chance do Caos deveria ser no máximo 95, recebido %d",
			chance,
		)
	}
}

func TestDungeonLegendaryPowerStaysBelowArcaneMarket(
	t *testing.T,
) {
	catalog, err := LoadCatalog()
	if err != nil {
		t.Fatalf(
			"erro carregando catálogo: %v",
			err,
		)
	}

	marketPrefixes :=
		[]string{
			"legendary_crystal_golem",
			"legendary_onyx_spider_queen",
			"legendary_eternal_labyrinth_minotaur",
			"legendary_jade_dragon",
		}

	typeLimits :=
		map[ItemType]int{}

	for _, item := range catalog.ItemsByRarity(
		RarityLegendary,
	) {

		isMarket := false

		for _, prefix := range marketPrefixes {

			if strings.HasPrefix(
				item.ID,
				prefix+"_",
			) {
				isMarket = true
				break
			}
		}

		if !isMarket {
			continue
		}

		current,
			exists :=
			typeLimits[item.Type]

		if !exists ||
			item.Power < current {

			typeLimits[item.Type] = item.Power
		}
	}

	for _, dungeon := range NormalDungeons() {

		for _, item := range catalog.ItemsByRarity(
			RarityLegendary,
		) {

			if !dungeonEquipmentAllowed(
				dungeon,
				item.ID,
			) {
				continue
			}

			marketMinimum,
				ok :=
				typeLimits[item.Type]

			if !ok {
				t.Fatalf(
					"sem mínimo de Mercado para tipo %s",
					item.Type,
				)
			}

			if item.Power >= marketMinimum {
				t.Fatalf(
					"%s: item %s (%d PC) invade faixa do Mercado Arcano (%d+)",
					dungeon.ID,
					item.ID,
					item.Power,
					marketMinimum,
				)
			}
		}
	}
}
