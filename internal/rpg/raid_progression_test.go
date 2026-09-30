package rpg

import "testing"

func TestRaidProgressionLegendaryPool(
	t *testing.T,
) {
	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	bosses :=
		RaidRotationCandidates(
			catalog,
			RarityLegendary,
			nil,
		)

	if len(bosses) != 2 {
		t.Fatalf(
			"esperados 2 Legendary ativos no Raid; encontrados %d",
			len(bosses),
		)
	}

	expected :=
		map[string]bool{
			"crimson_chimera": false,
			"ash_cerberus":    false,
		}

	for _, boss := range bosses {
		if _, exists :=
			expected[boss.ID]; !exists {

			t.Fatalf(
				"Legendary indevido na progressão Raid: %s",
				boss.ID,
			)
		}

		expected[boss.ID] = true
	}

	for bossID, found := range expected {

		if !found {
			t.Fatalf(
				"Legendary esperado não encontrado: %s",
				bossID,
			)
		}
	}
}

func TestRaidProgressionKeepsAllMythicAndSacred(
	t *testing.T,
) {
	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	for _, rarity := range []Rarity{
		RarityMythic,
		RaritySacred,
	} {

		all :=
			catalog.BossesByRarity(
				rarity,
			)

		active :=
			RaidRotationCandidates(
				catalog,
				rarity,
				nil,
			)

		if len(active) != len(all) {
			t.Fatalf(
				"%s: catálogo=%d ativo=%d",
				rarity,
				len(all),
				len(active),
			)
		}
	}
}

func TestRaidProgressionExcludesEarlierLegendarySources(
	t *testing.T,
) {
	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	excluded :=
		[]string{
			"black_dragon",
			"blood_moon_wolf",
			"abyssal_kraken",
			"crystal_golem",
			"onyx_spider_queen",
			"eternal_labyrinth_minotaur",
			"jade_dragon",
		}

	for _, bossID := range excluded {
		boss, exists :=
			catalog.BossByID(
				bossID,
			)

		if !exists {
			t.Fatalf(
				"Boss não encontrado: %s",
				bossID,
			)
		}

		if RaidBossEnabledForProgression(
			boss,
		) {
			t.Fatalf(
				"Boss %s não deveria estar no pool Raid",
				bossID,
			)
		}
	}
}
