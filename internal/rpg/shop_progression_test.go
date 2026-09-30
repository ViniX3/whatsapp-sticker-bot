package rpg

import "testing"

func TestShopEpicProgression(
	t *testing.T,
) {

	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	epics :=
		catalog.ItemsByRarity(
			RarityEpic,
		)

	if len(epics) != 50 {
		t.Fatalf(
			"catálogo Epic inesperado: %d itens; esperado 50",
			len(epics),
		)
	}

	eligible := 0

	weapons := 0
	shields := 0
	armors := 0

	blocked := 0

	for _, item := range epics {

		allowed :=
			shopEligible(
				item,
			)

		if item.Power <=
			ShopEpicMaxPower {

			if !allowed {
				t.Fatalf(
					"Epic %s com PC %d deveria estar disponível na Loja",
					item.ID,
					item.Power,
				)
			}

			eligible++

			switch item.Type {

			case ItemTypeWeapon:
				weapons++

			case ItemTypeShield:
				shields++

			case ItemTypeArmor:
				armors++

			default:
				t.Fatalf(
					"tipo inesperado no item %s: %s",
					item.ID,
					item.Type,
				)
			}

			continue
		}

		if allowed {
			t.Fatalf(
				"Epic %s com PC %d ultrapassa o teto %d e não deveria estar disponível",
				item.ID,
				item.Power,
				ShopEpicMaxPower,
			)
		}

		blocked++
	}

	if eligible != 19 {
		t.Fatalf(
			"Epics elegíveis = %d; esperado 19",
			eligible,
		)
	}

	if blocked != 31 {
		t.Fatalf(
			"Epics bloqueados = %d; esperado 31",
			blocked,
		)
	}

	if weapons != 12 {
		t.Fatalf(
			"armas Epic na Loja = %d; esperado 12",
			weapons,
		)
	}

	if shields != 6 {
		t.Fatalf(
			"escudos Epic na Loja = %d; esperado 6",
			shields,
		)
	}

	if armors != 1 {
		t.Fatalf(
			"armaduras Epic na Loja = %d; esperado 1",
			armors,
		)
	}
}

func TestShopEpicPowerCeiling(
	t *testing.T,
) {

	if ShopEpicMaxPower != 1000 {
		t.Fatalf(
			"ShopEpicMaxPower = %d; esperado 1000",
			ShopEpicMaxPower,
		)
	}
}
