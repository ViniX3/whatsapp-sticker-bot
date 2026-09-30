package rpg

import "testing"

func TestPVEBossAcceptsOnlyOwnBossDrop(
	t *testing.T,
) {
	enemy :=
		Enemy{
			ID: "regional_test_boss",

			Rarity: RarityLegendary,

			Boss: true,
		}

	ownDrop :=
		Item{
			ID: "regional_drop",

			Rarity: RarityLegendary,

			Source: SourceBossDrop,

			BossID: "regional_test_boss",
		}

	if !pveEquipmentEligibleForEnemy(
		enemy,
		ownDrop,
	) {
		t.Fatal(
			"Boss PvE deveria aceitar seu próprio BOSS_DROP",
		)
	}
}

func TestPVEBossRejectsRaidBossDrop(
	t *testing.T,
) {
	enemy :=
		Enemy{
			ID: "forest_boss_grande_lobo_branco",

			Rarity: RarityLegendary,

			Boss: true,
		}

	raidDrop :=
		Item{
			ID: "legendary_black_dragon_weapon",

			Rarity: RarityLegendary,

			Source: SourceBossDrop,

			BossID: "black_dragon",
		}

	if pveEquipmentEligibleForEnemy(
		enemy,
		raidDrop,
	) {
		t.Fatal(
			"Boss PvE não pode receber loot de Raid Boss",
		)
	}
}

func TestNormalPVERejectsBossDrop(
	t *testing.T,
) {
	enemy :=
		Enemy{
			ID: "normal_enemy",

			Rarity: RarityLegendary,

			Boss: false,
		}

	item :=
		Item{
			ID: "raid_item",

			Rarity: RarityLegendary,

			Source: SourceBossDrop,

			BossID: "jade_dragon",
		}

	if pveEquipmentEligibleForEnemy(
		enemy,
		item,
	) {
		t.Fatal(
			"inimigo comum não pode receber BOSS_DROP",
		)
	}
}

func TestRegionalPVEBossesCannotDropRaidEquipment(
	t *testing.T,
) {
	itemCatalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	enemyCatalog, err :=
		LoadEnemyCatalog()

	if err != nil {
		t.Fatal(err)
	}

	regionalBossIDs :=
		[]string{
			"forest_boss_grande_lobo_branco",
			"quarry_boss_colosso_primordial_de_granito",
			"mine_boss_tita_de_cristal_das_profundezas",
		}

	for _, bossID := range regionalBossIDs {

		boss, exists :=
			enemyCatalog.EnemyByID(
				bossID,
			)

		if !exists {
			t.Fatalf(
				"Boss regional não encontrado: %s",
				bossID,
			)
		}

		eligible :=
			0

		for _, item := range itemCatalog.Items() {

			if pveEquipmentEligibleForEnemy(
				boss,
				item,
			) {
				eligible++
			}
		}

		// Atualmente os Bosses regionais não possuem
		// equipamentos exclusivos próprios.
		if eligible != 0 {
			t.Fatalf(
				"Boss regional %s possui %d equipamentos elegíveis; esperado 0",
				boss.ID,
				eligible,
			)
		}
	}
}
