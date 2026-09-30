package rpg

import "testing"

func TestCatalogLoads(
	t *testing.T,
) {
	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatalf(
			"erro carregando catálogo RPG: %v",
			err,
		)
	}

	if len(
		catalog.Items(),
	) == 0 {

		t.Fatal(
			"catálogo RPG não possui itens",
		)
	}

	if len(
		catalog.Sets(),
	) == 0 {

		t.Fatal(
			"catálogo RPG não possui sets",
		)
	}
}

func TestSacredItems(
	t *testing.T,
) {
	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	items :=
		catalog.ItemsByRarity(
			RaritySacred,
		)

	if len(items) != 9 {
		t.Fatalf(
			"esperados 9 itens Sagrados, encontrados %d",
			len(items),
		)
	}

	for _, item := range items {

		if item.Source !=
			SourceBossDrop {

			t.Fatalf(
				"item Sagrado %s não é drop de Boss",
				item.ID,
			)
		}

		if !item.Unique {
			t.Fatalf(
				"item Sagrado %s deveria ser único",
				item.ID,
			)
		}

		if item.SetID == "" {
			t.Fatalf(
				"item Sagrado %s não possui set",
				item.ID,
			)
		}
	}
}

func TestSacredSets(
	t *testing.T,
) {
	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	sacredSets := 0

	for _, set := range catalog.Sets() {

		if set.Rarity !=
			RaritySacred {

			continue
		}

		sacredSets++

		if len(
			set.Pieces,
		) != 3 {

			t.Fatalf(
				"set Sagrado %s deveria possuir 3 peças",
				set.ID,
			)
		}
	}

	if sacredSets != 3 {
		t.Fatalf(
			"esperados 3 sets Sagrados, encontrados %d",
			sacredSets,
		)
	}
}

func TestOtherworldItems(
	t *testing.T,
) {
	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	items :=
		catalog.ItemsBySource(
			SourceOtherworldMerchant,
		)

	if len(items) != 12 {
		t.Fatalf(
			"esperados 12 itens do Mercador, encontrados %d",
			len(items),
		)
	}

	for _, item := range items {

		if item.Rarity ==
			RarityMythic ||
			item.Rarity ==
				RaritySacred {

			t.Fatalf(
				"Mercador não deveria vender item %s de raridade %s",
				item.ID,
				item.Rarity,
			)
		}

		if item.Currency !=
			CurrencyStellarStone {

			t.Fatalf(
				"item %s do Mercador deveria usar Pedras Estelares",
				item.ID,
			)
		}
	}
}

func TestBaseEquipmentCounts(
	t *testing.T,
) {
	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	expected :=
		map[Rarity]int{
			RarityWorn: 100,

			RarityCommon: 90,

			RarityRare: 80,

			RarityEpic: 50,
		}

	for rarity, expectedCount := range expected {

		items :=
			catalog.ItemsByRarity(
				rarity,
			)

		if len(items) !=
			expectedCount {

			t.Fatalf(
				"raridade %s: esperados %d itens, encontrados %d",
				rarity,
				expectedCount,
				len(items),
			)
		}
	}
}

func TestBaseEquipmentIsCraftable(
	t *testing.T,
) {
	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	rarities :=
		[]Rarity{
			RarityWorn,
			RarityCommon,
			RarityRare,
			RarityEpic,
		}

	for _, rarity := range rarities {

		items :=
			catalog.ItemsByRarity(
				rarity,
			)

		for _, item := range items {

			if item.Source !=
				SourceCraft {

				t.Fatalf(
					"item %s deveria ser CRAFT",
					item.ID,
				)
			}

			if len(
				item.CraftMaterials,
			) == 0 {

				t.Fatalf(
					"item %s não possui receita",
					item.ID,
				)
			}

			if item.Attack+
				item.Defense !=
				item.Power {

				t.Fatalf(
					"atributos inconsistentes no item %s",
					item.ID,
				)
			}
		}
	}
}

func TestFinalEquipmentCatalog(
	t *testing.T,
) {
	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	expected :=
		map[Rarity]int{
			RarityWorn: 100,

			RarityCommon: 90,

			RarityRare: 80,

			RarityEpic: 50,

			RarityLegendary: 62,

			RarityMythic: 46,

			RaritySacred: 9,
		}

	total := 0

	for rarity, expectedCount := range expected {

		items :=
			catalog.ItemsByRarity(
				rarity,
			)

		if len(items) !=
			expectedCount {

			t.Fatalf(
				"raridade %s: esperados %d itens, encontrados %d",
				rarity,
				expectedCount,
				len(items),
			)
		}

		total += len(items)
	}

	if total != 437 {
		t.Fatalf(
			"esperados 437 equipamentos, encontrados %d",
			total,
		)
	}
}

func TestLegendaryDistribution(
	t *testing.T,
) {
	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	items :=
		catalog.ItemsByRarity(
			RarityLegendary,
		)

	bossDrops := 0
	otherworld := 0

	for _, item := range items {

		switch item.Source {

		case SourceBossDrop:
			bossDrops++

		case SourceOtherworldMerchant:
			otherworld++
		}
	}

	if bossDrops != 50 {
		t.Fatalf(
			"esperados 50 Lendários de Boss, encontrados %d",
			bossDrops,
		)
	}

	if otherworld != 12 {
		t.Fatalf(
			"esperados 12 Lendários do Mercador, encontrados %d",
			otherworld,
		)
	}
}

func TestMythicDistribution(
	t *testing.T,
) {
	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	items :=
		catalog.ItemsByRarity(
			RarityMythic,
		)

	if len(items) != 46 {
		t.Fatalf(
			"esperados 46 itens Míticos, encontrados %d",
			len(items),
		)
	}

	for _, item := range items {

		if item.Source !=
			SourceBossDrop {

			t.Fatalf(
				"item Mítico %s não é drop de Boss",
				item.ID,
			)
		}

		if item.BossID == "" {
			t.Fatalf(
				"item Mítico %s não possui boss_id",
				item.ID,
			)
		}
	}
}

func TestHighTierSets(
	t *testing.T,
) {
	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	counts :=
		map[Rarity]int{}

	for _, set := range catalog.Sets() {

		counts[set.Rarity]++
	}

	if counts[RarityLegendary] != 20 {

		t.Fatalf(
			"esperados 12 sets Lendários, encontrados %d",
			counts[RarityLegendary],
		)
	}

	if counts[RarityMythic] != 15 {

		t.Fatalf(
			"esperados 9 sets Míticos, encontrados %d",
			counts[RarityMythic],
		)
	}

	if counts[RaritySacred] != 3 {

		t.Fatalf(
			"esperados 3 sets Sagrados, encontrados %d",
			counts[RaritySacred],
		)
	}
}
