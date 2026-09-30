package rpg

import "testing"

func TestMaterialCatalogLoads(
	t *testing.T,
) {
	catalog, err :=
		LoadMaterialCatalog()

	if err != nil {
		t.Fatalf(
			"erro carregando materiais: %v",
			err,
		)
	}

	if len(
		catalog.Materials(),
	) < 50 {

		t.Fatalf(
			"esperados pelo menos 50 materiais, encontrados %d",
			len(
				catalog.Materials(),
			),
		)
	}
}

func TestPhysicalMaterialsExist(
	t *testing.T,
) {
	catalog, err :=
		LoadMaterialCatalog()

	if err != nil {
		t.Fatal(err)
	}

	expected :=
		[]string{
			"wood_pine",
			"stone",
			"ingot_iron",
			"ingot_steel",
			"diamond",
			"ingot_mithril",
			"ingot_adamantite",
		}

	for _, materialID := range expected {

		if _, exists :=
			catalog.MaterialByID(
				materialID,
			); !exists {

			t.Fatalf(
				"material físico esperado não encontrado: %s",
				materialID,
			)
		}
	}
}

func TestMonsterMaterialsExist(
	t *testing.T,
) {
	catalog, err :=
		LoadMaterialCatalog()

	if err != nil {
		t.Fatal(err)
	}

	expected :=
		[]string{
			"ice_wyvern_claw",
			"giant_turtle_shell",
			"black_dragon_scale",
			"leviathan_fang",
			"ancient_demon_horn",
			"behemoth_shell",
			"titan_heart",
			"phoenix_feather",
		}

	for _, materialID := range expected {

		if _, exists :=
			catalog.MaterialByID(
				materialID,
			); !exists {

			t.Fatalf(
				"material de monstro esperado não encontrado: %s",
				materialID,
			)
		}
	}
}

func TestCraftRecipesAreValid(
	t *testing.T,
) {
	itemCatalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	materialCatalog, err :=
		LoadMaterialCatalog()

	if err != nil {
		t.Fatal(err)
	}

	if err :=
		ValidateCraftRecipes(
			itemCatalog,
			materialCatalog,
		); err != nil {

		t.Fatalf(
			"receita inválida: %v",
			err,
		)
	}
}
