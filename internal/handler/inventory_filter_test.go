package handler

import (
	"testing"

	"whatsapp-sticker-bot/internal/rpg"
)

func TestParseInventoryArguments(
	t *testing.T,
) {

	tests := []struct {
		args       []string
		wantKind   inventoryFilterKind
		wantRarity rpg.Rarity
		wantPage   int
	}{
		{
			args:     nil,
			wantKind: inventoryFilterAll,
			wantPage: 1,
		},
		{
			args:     []string{"2"},
			wantKind: inventoryFilterAll,
			wantPage: 2,
		},
		{
			args:     []string{"equipamentos"},
			wantKind: inventoryFilterEquipment,
			wantPage: 1,
		},
		{
			args:     []string{"materiais"},
			wantKind: inventoryFilterMaterial,
			wantPage: 1,
		},
		{
			args:     []string{"arma"},
			wantKind: inventoryFilterWeapon,
			wantPage: 1,
		},
		{
			args:     []string{"equipado"},
			wantKind: inventoryFilterEquipped,
			wantPage: 1,
		},
		{
			args:       []string{"ÉPICO"},
			wantKind:   inventoryFilterRarity,
			wantRarity: rpg.RarityEpic,
			wantPage:   1,
		},
		{
			args:       []string{"lendário", "2"},
			wantKind:   inventoryFilterRarity,
			wantRarity: rpg.RarityLegendary,
			wantPage:   2,
		},
	}

	for _, test := range tests {

		filter,
			page,
			err :=
			parseInventoryArguments(
				test.args,
			)

		if err != nil {
			t.Fatalf(
				"%v retornou erro: %v",
				test.args,
				err,
			)
		}

		if filter.Kind !=
			test.wantKind {

			t.Fatalf(
				"%v: filtro esperado %q, recebido %q",
				test.args,
				test.wantKind,
				filter.Kind,
			)
		}

		if filter.Rarity !=
			test.wantRarity {

			t.Fatalf(
				"%v: raridade esperada %q, recebida %q",
				test.args,
				test.wantRarity,
				filter.Rarity,
			)
		}

		if page !=
			test.wantPage {

			t.Fatalf(
				"%v: página esperada %d, recebida %d",
				test.args,
				test.wantPage,
				page,
			)
		}
	}
}

func TestParseInventoryArgumentsRejectsInvalid(
	t *testing.T,
) {

	tests := [][]string{
		{"qualquercoisa"},
		{"raro", "abc"},
		{"raro", "0"},
		{"arma", "2", "extra"},
	}

	for _, args := range tests {

		_,
			_,
			err :=
			parseInventoryArguments(
				args,
			)

		if err == nil {
			t.Fatalf(
				"%v deveria gerar erro",
				args,
			)
		}
	}
}

func TestInventoryViewMatchesFilter(
	t *testing.T,
) {

	weapon :=
		rpgInventoryView{
			Kind:        "Arma",
			Rarity:      rpg.RarityEpic,
			IsEquipment: true,
			Equipped:    true,
		}

	material :=
		rpgInventoryView{
			Kind:        "Metal",
			Rarity:      rpg.RarityRare,
			IsEquipment: false,
		}

	if !inventoryViewMatchesFilter(
		weapon,
		inventoryFilter{
			Kind: inventoryFilterWeapon,
		},
	) {
		t.Fatal(
			"arma deveria corresponder ao filtro weapon",
		)
	}

	if !inventoryViewMatchesFilter(
		weapon,
		inventoryFilter{
			Kind: inventoryFilterEquipped,
		},
	) {
		t.Fatal(
			"item equipado deveria corresponder ao filtro equipped",
		)
	}

	if !inventoryViewMatchesFilter(
		material,
		inventoryFilter{
			Kind: inventoryFilterMaterial,
		},
	) {
		t.Fatal(
			"material deveria corresponder ao filtro material",
		)
	}

	if inventoryViewMatchesFilter(
		material,
		inventoryFilter{
			Kind: inventoryFilterEquipment,
		},
	) {
		t.Fatal(
			"material não deveria corresponder a equipamento",
		)
	}
}
