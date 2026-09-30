package rpg

import "testing"

func TestLoadRaidBossCatalog(
	t *testing.T,
) {

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	bosses :=
		catalog.Bosses()

	if len(bosses) != 34 {
		t.Fatalf(
			"esperados 34 Raid Bosses, encontrados %d",
			len(bosses),
		)
	}

	counts :=
		map[Rarity]int{}

	for _, boss := range bosses {

		counts[boss.Rarity]++
	}

	if counts[RarityLegendary] != 16 {

		t.Fatalf(
			"esperados 16 Legendary, encontrados %d",
			counts[RarityLegendary],
		)
	}

	if counts[RarityMythic] != 15 {

		t.Fatalf(
			"esperados 15 Mythic, encontrados %d",
			counts[RarityMythic],
		)
	}

	if counts[RaritySacred] != 3 {

		t.Fatalf(
			"esperados 3 Sacred, encontrados %d",
			counts[RaritySacred],
		)
	}
}

func TestRaidBossCatalogPreservesExistingBosses(
	t *testing.T,
) {

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	expectedSetIDs :=
		[]string{
			"legendary_black_dragon",
			"legendary_leviathan",
			"legendary_ancient_demon",
			"legendary_behemoth",
			"legendary_storm_griffin",
			"legendary_frost_wyrm",
			"legendary_infernal_colossus",
			"legendary_blood_moon_wolf",

			"mythic_phoenix",
			"mythic_ancient_titan",
			"mythic_world_serpent",
			"mythic_celestial_hydra",
			"mythic_abyss_king",
			"mythic_void_colossus",
			"mythic_eternal_lich",
			"mythic_chaos_dragon",
			"mythic_moon_devourer",

			"sacred_bahamut",
			"sacred_azathor",
			"sacred_beyond_stars",
		}

	found :=
		make(
			map[string]bool,
		)

	for _, boss := range catalog.Bosses() {

		found[boss.SetID] = true
	}

	for _, setID := range expectedSetIDs {

		if !found[setID] {

			t.Fatalf(
				"Boss antigo perdido: set %s não encontrado",
				setID,
			)
		}
	}
}

func TestRaidBossLookup(
	t *testing.T,
) {

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	boss, exists :=
		catalog.BossByID(
			"  BEYOND_STARS ",
		)

	if !exists {
		t.Fatal(
			"Beyond Stars não encontrado",
		)
	}

	if boss.Rarity !=
		RaritySacred {

		t.Fatalf(
			"raridade esperada SACRED, recebida %s",
			boss.Rarity,
		)
	}

	if boss.SetID !=
		"sacred_beyond_stars" {

		t.Fatalf(
			"set inesperado: %s",
			boss.SetID,
		)
	}
}

func TestRaidBossPowerProgression(
	t *testing.T,
) {

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	for _, rarity := range []Rarity{
		RarityLegendary,
		RarityMythic,
		RaritySacred,
	} {

		bosses :=
			catalog.
				BossesByRarity(
					rarity,
				)

		previous := 0

		for _, boss := range bosses {

			if boss.Power <=
				previous {

				t.Fatalf(
					"%s fora de ordem: %s possui %d após %d",
					rarity,
					boss.ID,
					boss.Power,
					previous,
				)
			}

			previous =
				boss.Power
		}
	}
}

func TestRaidBossImageKeys(
	t *testing.T,
) {

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	seen :=
		make(
			map[string]bool,
		)

	for _, boss := range catalog.Bosses() {

		if boss.ImageKey == "" {
			t.Fatalf(
				"Boss %s sem image_key",
				boss.ID,
			)
		}

		if seen[boss.ImageKey] {

			t.Fatalf(
				"image_key duplicado: %s",
				boss.ImageKey,
			)
		}

		seen[boss.ImageKey] = true
	}
}
