package rpg

import (
	"errors"
	"strings"
)

// ResolveEquipmentReference aceita:
//
//	common_armor_030
//	L190
//	R012
//
// O restante do RPG continua trabalhando com o ID
// interno real do equipamento.
func ResolveEquipmentReference(
	catalog *Catalog,
	reference string,
) (
	Item,
	error,
) {
	if catalog == nil {
		return Item{},
			ErrItemNotFound
	}

	reference =
		strings.TrimSpace(
			reference,
		)

	if reference == "" {
		return Item{},
			ErrItemNotFound
	}

	if item, exists :=
		catalog.ItemByID(
			reference,
		); exists {

		return item, nil
	}

	offer, err :=
		FindShopOffer(
			catalog,
			reference,
		)

	if err == nil {
		return offer.Item,
			nil
	}

	if !errors.Is(
		err,
		ErrShopOfferNotFound,
	) {
		return Item{},
			err
	}

	recipe, err :=
		FindForgeRecipe(
			catalog,
			reference,
		)

	if err == nil {
		return recipe.Item,
			nil
	}

	if !errors.Is(
		err,
		ErrForgeRecipeNotFound,
	) {
		return Item{},
			err
	}

	return Item{},
		ErrItemNotFound
}
