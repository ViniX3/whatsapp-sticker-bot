package handler

import (
	"fmt"
	"strconv"

	"whatsapp-sticker-bot/internal/rpg"
)

type inventoryFilterKind string

const (
	inventoryFilterAll       inventoryFilterKind = ""
	inventoryFilterEquipment inventoryFilterKind = "equipment"
	inventoryFilterMaterial  inventoryFilterKind = "material"
	inventoryFilterWeapon    inventoryFilterKind = "weapon"
	inventoryFilterShield    inventoryFilterKind = "shield"
	inventoryFilterArmor     inventoryFilterKind = "armor"
	inventoryFilterEquipped  inventoryFilterKind = "equipped"
	inventoryFilterRarity    inventoryFilterKind = "rarity"
)

type inventoryFilter struct {
	Kind   inventoryFilterKind
	Rarity rpg.Rarity
	Label  string
	Token  string
}

func parseInventoryArguments(
	args []string,
) (
	inventoryFilter,
	int,
	error,
) {

	filter := inventoryFilter{
		Kind: inventoryFilterAll,
	}

	page := 1

	if len(args) == 0 {
		return filter,
			page,
			nil
	}

	if len(args) > 2 {
		return filter,
			0,
			fmt.Errorf(
				"argumentos demais",
			)
	}

	// Mantém compatibilidade:
	//
	// !inventario 2
	if len(args) == 1 {
		if parsedPage,
			err :=
			strconv.Atoi(
				args[0],
			); err == nil {

			if parsedPage <= 0 {
				return filter,
					0,
					fmt.Errorf(
						"página inválida",
					)
			}

			return filter,
				parsedPage,
				nil
		}
	}

	parsedFilter,
		ok :=
		parseInventoryFilter(
			args[0],
		)

	if !ok {
		return filter,
			0,
			fmt.Errorf(
				"filtro inválido",
			)
	}

	filter =
		parsedFilter

	if len(args) == 2 {
		parsedPage,
			err :=
			strconv.Atoi(
				args[1],
			)

		if err != nil ||
			parsedPage <= 0 {

			return filter,
				0,
				fmt.Errorf(
					"página inválida",
				)
		}

		page =
			parsedPage
	}

	return filter,
		page,
		nil
}

func parseInventoryFilter(
	value string,
) (
	inventoryFilter,
	bool,
) {

	normalized :=
		normalizeCommandToken(
			value,
		)

	switch normalized {

	case "equipamento",
		"equipamentos",
		"gear":

		return inventoryFilter{
				Kind:  inventoryFilterEquipment,
				Label: "EQUIPAMENTOS",
				Token: "equipamentos",
			},
			true

	case "material",
		"materiais",
		"recursos":

		return inventoryFilter{
				Kind:  inventoryFilterMaterial,
				Label: "MATERIAIS",
				Token: "materiais",
			},
			true

	case "arma",
		"armas",
		"weapon",
		"weapons":

		return inventoryFilter{
				Kind:  inventoryFilterWeapon,
				Label: "ARMAS",
				Token: "arma",
			},
			true

	case "escudo",
		"escudos",
		"shield",
		"shields":

		return inventoryFilter{
				Kind:  inventoryFilterShield,
				Label: "ESCUDOS",
				Token: "escudo",
			},
			true

	case "armadura",
		"armaduras",
		"armor",
		"armors":

		return inventoryFilter{
				Kind:  inventoryFilterArmor,
				Label: "ARMADURAS",
				Token: "armadura",
			},
			true

	case "equipado",
		"equipados",
		"equipada",
		"equipadas":

		return inventoryFilter{
				Kind:  inventoryFilterEquipped,
				Label: "EQUIPADOS",
				Token: "equipado",
			},
			true
	}

	rarity,
		ok :=
		parseInventoryRarity(
			normalized,
		)

	if ok {
		return inventoryFilter{
				Kind:   inventoryFilterRarity,
				Rarity: rarity,
				Label:  rpgRarityName(rarity),
				Token:  normalized,
			},
			true
	}

	return inventoryFilter{},
		false
}

func parseInventoryRarity(
	value string,
) (
	rpg.Rarity,
	bool,
) {

	switch normalizeCommandToken(
		value,
	) {

	case "desgastado",
		"worn":
		return rpg.RarityWorn,
			true

	case "comum",
		"common":
		return rpg.RarityCommon,
			true

	case "raro",
		"rare":
		return rpg.RarityRare,
			true

	case "epico",
		"epic":
		return rpg.RarityEpic,
			true

	case "lendario",
		"legendary":
		return rpg.RarityLegendary,
			true

	case "mitico",
		"mythic":
		return rpg.RarityMythic,
			true

	case "sagrado",
		"sacred":
		return rpg.RaritySacred,
			true
	}

	return "",
		false
}

func inventoryViewMatchesFilter(
	view rpgInventoryView,
	filter inventoryFilter,
) bool {

	switch filter.Kind {

	case inventoryFilterAll:
		return true

	case inventoryFilterEquipment:
		return view.IsEquipment

	case inventoryFilterMaterial:
		return !view.IsEquipment

	case inventoryFilterWeapon:
		return view.IsEquipment &&
			normalizeCommandToken(
				view.Kind,
			) == "arma"

	case inventoryFilterShield:
		return view.IsEquipment &&
			normalizeCommandToken(
				view.Kind,
			) == "escudo"

	case inventoryFilterArmor:
		return view.IsEquipment &&
			normalizeCommandToken(
				view.Kind,
			) == "armadura"

	case inventoryFilterEquipped:
		return view.IsEquipment &&
			view.Equipped

	case inventoryFilterRarity:
		return view.Rarity ==
			filter.Rarity
	}

	return false
}

func applyInventoryFilter(
	views []rpgInventoryView,
	filter inventoryFilter,
) []rpgInventoryView {

	if filter.Kind ==
		inventoryFilterAll {

		return views
	}

	filtered :=
		make(
			[]rpgInventoryView,
			0,
			len(views),
		)

	for _, view := range views {

		if inventoryViewMatchesFilter(
			view,
			filter,
		) {

			filtered =
				append(
					filtered,
					view,
				)
		}
	}

	return filtered
}
