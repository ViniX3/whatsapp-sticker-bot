package rpg

import "testing"

func TestEnemyCatalogDistribution(
	t *testing.T,
) {
	catalog, err :=
		LoadEnemyCatalog()

	if err != nil {
		t.Fatalf(
			"LoadEnemyCatalog(): %v",
			err,
		)
	}

	enemies :=
		catalog.Enemies()

	if len(enemies) != 93 {
		t.Fatalf(
			"esperado 93 monstros, recebido %d",
			len(enemies),
		)
	}

	regions :=
		[]GatheringRegion{
			GatheringForest,
			GatheringQuarry,
			GatheringMine,
		}

	for _, region := range regions {

		if got :=
			len(
				catalog.EnemiesByRegionAndRarity(
					region,
					RarityCommon,
				),
			); got != 15 {

			t.Fatalf(
				"%s: esperado 15 comuns, recebido %d",
				region,
				got,
			)
		}

		if got :=
			len(
				catalog.EnemiesByRegionAndRarity(
					region,
					RarityRare,
				),
			); got != 10 {

			t.Fatalf(
				"%s: esperado 10 raros, recebido %d",
				region,
				got,
			)
		}

		if got :=
			len(
				catalog.EnemiesByRegionAndRarity(
					region,
					RarityEpic,
				),
			); got != 5 {

			t.Fatalf(
				"%s: esperado 5 épicos, recebido %d",
				region,
				got,
			)
		}

		boss, exists :=
			catalog.BossByRegion(
				region,
			)

		if !exists {
			t.Fatalf(
				"%s: boss não encontrado",
				region,
			)
		}

		if boss.Rarity !=
			RarityLegendary {

			t.Fatalf(
				"%s: boss não é lendário",
				region,
			)
		}

		if boss.Blessing == nil {
			t.Fatalf(
				"%s: boss sem bênção",
				region,
			)
		}
	}
}

func TestEnemyRegionalBosses(
	t *testing.T,
) {
	catalog, err :=
		LoadEnemyCatalog()

	if err != nil {
		t.Fatal(err)
	}

	tests :=
		map[GatheringRegion]string{
			GatheringForest: "Grande Lobo Branco",

			GatheringQuarry: "Colosso Primordial de Granito",

			GatheringMine: "Titã de Cristal das Profundezas",
		}

	for region, expectedName := range tests {

		boss, exists :=
			catalog.BossByRegion(
				region,
			)

		if !exists {
			t.Fatalf(
				"boss ausente em %s",
				region,
			)
		}

		if boss.Name !=
			expectedName {

			t.Fatalf(
				"%s: esperado %q, recebido %q",
				region,
				expectedName,
				boss.Name,
			)
		}
	}
}
