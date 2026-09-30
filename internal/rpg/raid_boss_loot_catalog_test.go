package rpg

import "testing"

func TestEveryRaidBossHasThematicSet(
	t *testing.T,
) {
	bossCatalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	itemCatalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	bosses :=
		bossCatalog.Bosses()

	if len(bosses) != 34 {
		t.Fatalf(
			"esperados 34 Raid Bosses, encontrados %d",
			len(bosses),
		)
	}

	for _, boss := range bosses {

		set, exists :=
			itemCatalog.SetByID(
				boss.SetID,
			)

		if !exists {
			t.Fatalf(
				"Boss %s não possui set %s",
				boss.ID,
				boss.SetID,
			)
		}

		if set.Rarity !=
			boss.Rarity {

			t.Fatalf(
				"set %s possui raridade %s; Boss é %s",
				set.ID,
				set.Rarity,
				boss.Rarity,
			)
		}

		if len(
			set.Pieces,
		) != 3 {

			t.Fatalf(
				"set %s deve possuir 3 peças; possui %d",
				set.ID,
				len(set.Pieces),
			)
		}

		for _, itemID := range set.Pieces {

			item, exists :=
				itemCatalog.ItemByID(
					itemID,
				)

			if !exists {
				t.Fatalf(
					"set %s referencia item inexistente %s",
					set.ID,
					itemID,
				)
			}

			if item.SetID !=
				boss.SetID {

				t.Fatalf(
					"item %s aponta para set %s; esperado %s",
					item.ID,
					item.SetID,
					boss.SetID,
				)
			}

			if item.Rarity !=
				boss.Rarity {

				t.Fatalf(
					"item %s possui raridade %s; esperado %s",
					item.ID,
					item.Rarity,
					boss.Rarity,
				)
			}

			if item.Source !=
				SourceBossDrop {

				t.Fatalf(
					"item %s não é BOSS_DROP",
					item.ID,
				)
			}

			if item.BossID !=
				boss.ID {

				t.Fatalf(
					"item %s possui boss_id %s; esperado %s",
					item.ID,
					item.BossID,
					boss.ID,
				)
			}

		}
	}
}
