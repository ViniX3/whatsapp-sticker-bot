package rpg

import (
	"database/sql"
	"errors"
	"fmt"

	"whatsapp-sticker-bot/internal/database"
)

var (
	ErrItemNotFound = errors.New(
		"item não encontrado no catálogo",
	)

	ErrItemNotCraftable = errors.New(
		"item não pode ser fabricado",
	)

	ErrUniqueItemOwned = errors.New(
		"jogador já possui este item único",
	)
)

type MissingMaterial struct {
	MaterialID string

	Required int

	Owned int
}

type MissingMaterialsError struct {
	Materials []MissingMaterial
}

func (
	err *MissingMaterialsError,
) Error() string {
	return "materiais insuficientes para fabricar o item"
}

type CraftCheck struct {
	CanCraft bool

	Missing []MissingMaterial
}

type CraftResult struct {
	Item Item

	Quantity int
}

func ValidateCraftRecipes(
	catalog *Catalog,
	materials *MaterialCatalog,
) error {
	for _, item := range catalog.Items() {

		if item.Source !=
			SourceCraft {

			if len(
				item.CraftMaterials,
			) > 0 {

				return fmt.Errorf(
					"item %s possui receita mas não é CRAFT",
					item.ID,
				)
			}

			continue
		}

		if len(
			item.CraftMaterials,
		) == 0 {

			return fmt.Errorf(
				"item %s é CRAFT mas não possui materiais",
				item.ID,
			)
		}

		seen :=
			make(
				map[string]bool,
			)

		for _, requirement := range item.CraftMaterials {

			if seen[requirement.MaterialID] {
				return fmt.Errorf(
					"item %s possui material duplicado na receita: %s",
					item.ID,
					requirement.MaterialID,
				)
			}

			seen[requirement.MaterialID] = true

			if _, exists :=
				materials.MaterialByID(
					requirement.MaterialID,
				); !exists {

				return fmt.Errorf(
					"item %s referencia material inexistente: %s",
					item.ID,
					requirement.MaterialID,
				)
			}
		}
	}

	return nil
}

func CheckCraftRequirements(
	groupJID string,
	jid string,
	itemID string,
	catalog *Catalog,
	materials *MaterialCatalog,
) (*CraftCheck, error) {
	item, exists :=
		catalog.ItemByID(
			itemID,
		)

	if !exists {
		return nil,
			ErrItemNotFound
	}

	if item.Source !=
		SourceCraft ||
		len(
			item.CraftMaterials,
		) == 0 {

		return nil,
			ErrItemNotCraftable
	}

	if err :=
		ValidateCraftRecipes(
			catalog,
			materials,
		); err != nil {

		return nil, err
	}

	check :=
		&CraftCheck{
			CanCraft: true,

			Missing: make(
				[]MissingMaterial,
				0,
			),
		}

	for _, requirement := range item.CraftMaterials {

		quantity, err :=
			GetInventoryQuantity(
				groupJID,
				jid,
				requirement.MaterialID,
			)

		if err != nil {
			return nil, err
		}

		if quantity >=
			requirement.Quantity {

			continue
		}

		check.CanCraft =
			false

		check.Missing =
			append(
				check.Missing,
				MissingMaterial{
					MaterialID: requirement.MaterialID,

					Required: requirement.Quantity,

					Owned: quantity,
				},
			)
	}

	return check, nil
}

func CraftItem(
	groupJID string,
	jid string,
	itemID string,
	catalog *Catalog,
	materials *MaterialCatalog,
) (*CraftResult, error) {
	item, exists :=
		catalog.ItemByID(
			itemID,
		)

	if !exists {
		return nil,
			ErrItemNotFound
	}

	if item.Source !=
		SourceCraft ||
		len(
			item.CraftMaterials,
		) == 0 {

		return nil,
			ErrItemNotCraftable
	}

	if err :=
		ValidateCraftRecipes(
			catalog,
			materials,
		); err != nil {

		return nil, err
	}

	if err :=
		ensureSchema(); err != nil {

		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando crafting: %w",
				err,
			)
	}

	defer tx.Rollback()

	if _, err :=
		ensurePlayerTx(
			tx,
			groupJID,
			jid,
		); err != nil {

		return nil, err
	}

	if item.Unique {
		var owned int

		err :=
			tx.QueryRow(`
				SELECT quantity
				FROM rpg_inventory
				WHERE group_jid = ?
				  AND jid = ?
				  AND item_id = ?
			`,
				groupJID,
				jid,
				item.ID,
			).Scan(
				&owned,
			)

		if err == nil &&
			owned > 0 {

			return nil,
				ErrUniqueItemOwned
		}

		if err != nil &&
			!errors.Is(
				err,
				sql.ErrNoRows,
			) {

			return nil,
				fmt.Errorf(
					"erro consultando item único: %w",
					err,
				)
		}
	}

	missing :=
		make(
			[]MissingMaterial,
			0,
		)

	ownedQuantities :=
		make(
			map[string]int,
		)

	for _, requirement := range item.CraftMaterials {

		var owned int

		err :=
			tx.QueryRow(`
				SELECT quantity
				FROM rpg_inventory
				WHERE group_jid = ?
				  AND jid = ?
				  AND item_id = ?
			`,
				groupJID,
				jid,
				requirement.MaterialID,
			).Scan(
				&owned,
			)

		if errors.Is(
			err,
			sql.ErrNoRows,
		) {
			owned = 0

		} else if err != nil {
			return nil,
				fmt.Errorf(
					"erro consultando material %s: %w",
					requirement.MaterialID,
					err,
				)
		}

		ownedQuantities[requirement.MaterialID] = owned

		if owned <
			requirement.Quantity {

			missing =
				append(
					missing,
					MissingMaterial{
						MaterialID: requirement.MaterialID,

						Required: requirement.Quantity,

						Owned: owned,
					},
				)
		}
	}

	if len(missing) > 0 {
		return nil,
			&MissingMaterialsError{
				Materials: missing,
			}
	}

	for _, requirement := range item.CraftMaterials {

		owned :=
			ownedQuantities[requirement.MaterialID]

		remaining :=
			owned -
				requirement.Quantity

		if remaining == 0 {
			_, err =
				tx.Exec(`
					DELETE FROM rpg_inventory
					WHERE group_jid = ?
					  AND jid = ?
					  AND item_id = ?
				`,
					groupJID,
					jid,
					requirement.MaterialID,
				)

		} else {
			_, err =
				tx.Exec(`
					UPDATE rpg_inventory
					SET
						quantity = ?,
						updated_at =
							CURRENT_TIMESTAMP
					WHERE group_jid = ?
					  AND jid = ?
					  AND item_id = ?
				`,
					remaining,
					groupJID,
					jid,
					requirement.MaterialID,
				)
		}

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro consumindo material %s: %w",
					requirement.MaterialID,
					err,
				)
		}
	}

	_, err =
		tx.Exec(`
			INSERT INTO rpg_inventory (
				group_jid,
				jid,
				item_id,
				quantity
			)
			VALUES (?, ?, ?, 1)

			ON CONFLICT (
				group_jid,
				jid,
				item_id
			)
			DO UPDATE SET
				quantity =
					rpg_inventory.quantity + 1,

				updated_at =
					CURRENT_TIMESTAMP
		`,
			groupJID,
			jid,
			item.ID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro adicionando item fabricado ao inventário: %w",
				err,
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando crafting: %w",
				err,
			)
	}

	return &CraftResult{
		Item: item,

		Quantity: 1,
	}, nil
}
