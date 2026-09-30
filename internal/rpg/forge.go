package rpg

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"whatsapp-sticker-bot/internal/database"
)

const (
	ForgeRareMinGold = 100000
	ForgeRareMaxGold = 400000

	ForgeEpicMinGold = 500000
	ForgeEpicMaxGold = 2000000
)

var (
	ErrForgeRecipeNotFound = errors.New(
		"receita da forja não encontrada",
	)

	ErrForgeInsufficientGold = errors.New(
		"Gold insuficiente para forjar",
	)

	ErrForgeRarityNotAllowed = errors.New(
		"raridade não pode ser criada na forja",
	)
)

type ForgeGoldError struct {
	Required int
	Owned    int
}

func (err *ForgeGoldError) Error() string {
	return fmt.Sprintf(
		"%s: necessário %d, disponível %d",
		ErrForgeInsufficientGold,
		err.Required,
		err.Owned,
	)
}

func (err *ForgeGoldError) Unwrap() error {
	return ErrForgeInsufficientGold
}

type ForgeRecipe struct {
	Code string

	Item Item

	GoldCost int
}

type ForgeMaterialStatus struct {
	Material Material

	Required int
	Owned    int
}

type ForgeCheck struct {
	Recipe ForgeRecipe

	GoldOwned int

	Materials []ForgeMaterialStatus

	UniqueAlreadyOwned bool

	CanForge bool
}

type ForgeResult struct {
	Recipe ForgeRecipe

	Quantity int

	GoldBalance int
}

// InitForge cria somente a infraestrutura persistente
// de identificação das receitas.
//
// Os equipamentos continuam sendo definidos no catálogo
// JSON, usando SourceCraft + CraftMaterials.
func InitForge(
	catalog *Catalog,
) error {
	if err := ensureForgeSchema(); err != nil {
		return err
	}

	return syncForgeRecipes(
		catalog,
	)
}

func ensureForgeSchema() error {
	if database.DB == nil {
		return errors.New(
			"banco de dados não inicializado",
		)
	}

	_, err :=
		database.DB.Exec(`
			CREATE TABLE IF NOT EXISTS
				rpg_forge_recipes (
					item_id TEXT PRIMARY KEY,

					recipe_code TEXT NOT NULL UNIQUE,

					rarity TEXT NOT NULL,

					created_at DATETIME NOT NULL
						DEFAULT CURRENT_TIMESTAMP,

					updated_at DATETIME NOT NULL
						DEFAULT CURRENT_TIMESTAMP
				);

			CREATE INDEX IF NOT EXISTS
				idx_rpg_forge_recipes_code
			ON rpg_forge_recipes (
				recipe_code
			);
		`)

	if err != nil {
		return fmt.Errorf(
			"erro criando estrutura da forja: %w",
			err,
		)
	}

	return nil
}

func forgeEligible(
	item Item,
) bool {
	if item.Source !=
		SourceCraft {

		return false
	}

	if len(
		item.CraftMaterials,
	) == 0 {

		return false
	}

	switch item.Rarity {
	case RarityRare,
		RarityEpic:

		return true
	}

	return false
}

// ForgeGoldCost calcula o custo em Gold.
//
// O custo cresce com o Poder do item, mas fica
// limitado à faixa econômica definida para cada tier.
//
// Raro:
//
//	100k até 400k
//
// Épico:
//
//	500k até 2M
func ForgeGoldCost(
	item Item,
) (
	int,
	error,
) {
	switch item.Rarity {
	case RarityRare:
		cost :=
			ForgeRareMinGold +
				item.Power*750

		return clampForgeGold(
			cost,
			ForgeRareMinGold,
			ForgeRareMaxGold,
		), nil

	case RarityEpic:
		cost :=
			ForgeEpicMinGold +
				item.Power*1500

		return clampForgeGold(
			cost,
			ForgeEpicMinGold,
			ForgeEpicMaxGold,
		), nil

	default:
		return 0,
			ErrForgeRarityNotAllowed
	}
}

func clampForgeGold(
	value int,
	minimum int,
	maximum int,
) int {
	if value < minimum {
		return minimum
	}

	if value > maximum {
		return maximum
	}

	return value
}

func forgeRecipePrefix(
	rarity Rarity,
) (
	string,
	error,
) {
	switch rarity {
	case RarityRare:
		return "R", nil

	case RarityEpic:
		return "E", nil
	}

	return "",
		ErrForgeRarityNotAllowed
}

func forgeCandidateItems(
	catalog *Catalog,
) (
	[]Item,
	error,
) {
	if catalog == nil {
		return nil,
			errors.New(
				"catálogo RPG não inicializado",
			)
	}

	items :=
		make(
			[]Item,
			0,
		)

	for _, item := range catalog.Items() {

		if !forgeEligible(
			item,
		) {
			continue
		}

		items =
			append(
				items,
				item,
			)
	}

	sort.Slice(
		items,
		func(
			i int,
			j int,
		) bool {
			if items[i].Rarity !=
				items[j].Rarity {

				return items[i].
					Rarity.
					Rank() <
					items[j].
						Rarity.
						Rank()
			}

			return items[i].ID <
				items[j].ID
		},
	)

	return items, nil
}

// syncForgeRecipes garante que cada item fabricável
// possua um código persistente.
//
// Exemplo:
//
//	R001
//	R002
//	E001
//
// Quando novos equipamentos forem adicionados ao catálogo,
// os códigos antigos NÃO mudam.
func syncForgeRecipes(
	catalog *Catalog,
) error {
	if err :=
		ensureForgeSchema(); err != nil {

		return err
	}

	items, err :=
		forgeCandidateItems(
			catalog,
		)

	if err != nil {
		return err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return fmt.Errorf(
			"erro iniciando sincronização da forja: %w",
			err,
		)
	}

	defer tx.Rollback()

	rows, err :=
		tx.Query(`
			SELECT
				item_id,
				recipe_code
			FROM rpg_forge_recipes
		`)

	if err != nil {
		return fmt.Errorf(
			"erro consultando receitas da forja: %w",
			err,
		)
	}

	existing :=
		make(
			map[string]string,
		)

	usedCodes :=
		make(
			map[string]bool,
		)

	maxCode :=
		map[string]int{
			"R": 0,
			"E": 0,
		}

	for rows.Next() {
		var itemID string
		var code string

		if err :=
			rows.Scan(
				&itemID,
				&code,
			); err != nil {

			rows.Close()

			return fmt.Errorf(
				"erro lendo receita da forja: %w",
				err,
			)
		}

		existing[itemID] =
			code

		usedCodes[strings.ToUpper(
			code,
		)] = true

		upper :=
			strings.ToUpper(
				code,
			)

		if len(upper) < 2 {
			continue
		}

		prefix :=
			upper[:1]

		number, parseErr :=
			strconv.Atoi(
				upper[1:],
			)

		if parseErr != nil {
			continue
		}

		if number >
			maxCode[prefix] {

			maxCode[prefix] =
				number
		}
	}

	if err :=
		rows.Err(); err != nil {

		rows.Close()

		return fmt.Errorf(
			"erro iterando receitas da forja: %w",
			err,
		)
	}

	if err :=
		rows.Close(); err != nil {

		return fmt.Errorf(
			"erro fechando consulta da forja: %w",
			err,
		)
	}

	for _, item := range items {

		if code, exists :=
			existing[item.ID]; exists {

			_, err =
				tx.Exec(`
					UPDATE rpg_forge_recipes
					SET
						rarity = ?,
						updated_at =
							CURRENT_TIMESTAMP
					WHERE item_id = ?
				`,
					string(
						item.Rarity,
					),
					item.ID,
				)

			if err != nil {
				return fmt.Errorf(
					"erro atualizando receita %s: %w",
					code,
					err,
				)
			}

			continue
		}

		prefix, err :=
			forgeRecipePrefix(
				item.Rarity,
			)

		if err != nil {
			return err
		}

		for {
			maxCode[prefix]++

			code :=
				fmt.Sprintf(
					"%s%03d",
					prefix,
					maxCode[prefix],
				)

			if usedCodes[code] {
				continue
			}

			_, err =
				tx.Exec(`
					INSERT INTO
						rpg_forge_recipes (
							item_id,
							recipe_code,
							rarity
						)
					VALUES (?, ?, ?)
				`,
					item.ID,
					code,
					string(
						item.Rarity,
					),
				)

			if err != nil {
				return fmt.Errorf(
					"erro registrando receita %s: %w",
					code,
					err,
				)
			}

			existing[item.ID] =
				code

			usedCodes[code] =
				true

			break
		}
	}

	if err :=
		tx.Commit(); err != nil {

		return fmt.Errorf(
			"erro confirmando receitas da forja: %w",
			err,
		)
	}

	return nil
}

func ListForgeRecipes(
	catalog *Catalog,
	rarity Rarity,
) (
	[]ForgeRecipe,
	error,
) {
	if rarity != "" &&
		rarity != RarityRare &&
		rarity != RarityEpic {

		return nil,
			ErrForgeRarityNotAllowed
	}

	if err :=
		syncForgeRecipes(
			catalog,
		); err != nil {

		return nil, err
	}

	rows, err :=
		database.DB.Query(`
			SELECT
				item_id,
				recipe_code
			FROM rpg_forge_recipes
			ORDER BY recipe_code ASC
		`)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro listando receitas da forja: %w",
				err,
			)
	}

	defer rows.Close()

	recipes :=
		make(
			[]ForgeRecipe,
			0,
		)

	for rows.Next() {
		var itemID string
		var code string

		if err :=
			rows.Scan(
				&itemID,
				&code,
			); err != nil {

			return nil,
				fmt.Errorf(
					"erro lendo receita da forja: %w",
					err,
				)
		}

		item, exists :=
			catalog.ItemByID(
				itemID,
			)

		if !exists ||
			!forgeEligible(
				item,
			) {

			continue
		}

		if rarity != "" &&
			item.Rarity != rarity {

			continue
		}

		cost, err :=
			ForgeGoldCost(
				item,
			)

		if err != nil {
			return nil, err
		}

		recipes =
			append(
				recipes,
				ForgeRecipe{
					Code: code,

					Item: item,

					GoldCost: cost,
				},
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"erro iterando receitas da forja: %w",
				err,
			)
	}

	sort.Slice(
		recipes,
		func(
			i int,
			j int,
		) bool {
			if recipes[i].
				Item.
				Rarity !=
				recipes[j].
					Item.
					Rarity {

				return recipes[i].
					Item.
					Rarity.
					Rank() <
					recipes[j].
						Item.
						Rarity.
						Rank()
			}

			return recipes[i].Code <
				recipes[j].Code
		},
	)

	return recipes, nil
}

func FindForgeRecipe(
	catalog *Catalog,
	reference string,
) (
	ForgeRecipe,
	error,
) {
	reference =
		strings.TrimSpace(
			reference,
		)

	if reference == "" {
		return ForgeRecipe{},
			ErrForgeRecipeNotFound
	}

	recipes, err :=
		ListForgeRecipes(
			catalog,
			"",
		)

	if err != nil {
		return ForgeRecipe{},
			err
	}

	for _, recipe := range recipes {

		if strings.EqualFold(
			recipe.Code,
			reference,
		) ||
			strings.EqualFold(
				recipe.Item.ID,
				reference,
			) {

			return recipe, nil
		}
	}

	return ForgeRecipe{},
		ErrForgeRecipeNotFound
}

func CheckForgeRequirements(
	groupJID string,
	jid string,
	reference string,
	catalog *Catalog,
	materials *MaterialCatalog,
) (
	*ForgeCheck,
	error,
) {
	if materials == nil {
		return nil,
			errors.New(
				"catálogo de materiais não inicializado",
			)
	}

	if err :=
		ValidateCraftRecipes(
			catalog,
			materials,
		); err != nil {

		return nil, err
	}

	recipe, err :=
		FindForgeRecipe(
			catalog,
			reference,
		)

	if err != nil {
		return nil, err
	}

	if err :=
		ensureSchema(); err != nil {

		return nil, err
	}

	var gold int

	err =
		database.DB.QueryRow(`
			SELECT gold
			FROM group_wallets
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&gold,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			ErrWalletNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando Gold para forja: %w",
				err,
			)
	}

	check :=
		&ForgeCheck{
			Recipe: recipe,

			GoldOwned: gold,

			Materials: make(
				[]ForgeMaterialStatus,
				0,
				len(
					recipe.Item.
						CraftMaterials,
				),
			),

			CanForge: gold >=
				recipe.GoldCost,
		}

	if recipe.Item.Unique {
		owned, err :=
			GetInventoryQuantity(
				groupJID,
				jid,
				recipe.Item.ID,
			)

		if err != nil {
			return nil, err
		}

		if owned > 0 {
			check.UniqueAlreadyOwned =
				true

			check.CanForge =
				false
		}
	}

	for _, requirement := range recipe.Item.
		CraftMaterials {

		material, exists :=
			materials.MaterialByID(
				requirement.MaterialID,
			)

		if !exists {
			return nil,
				fmt.Errorf(
					"material inexistente na receita: %s",
					requirement.MaterialID,
				)
		}

		owned, err :=
			GetInventoryQuantity(
				groupJID,
				jid,
				requirement.MaterialID,
			)

		if err != nil {
			return nil, err
		}

		if owned <
			requirement.Quantity {

			check.CanForge =
				false
		}

		check.Materials =
			append(
				check.Materials,
				ForgeMaterialStatus{
					Material: material,

					Required: requirement.
						Quantity,

					Owned: owned,
				},
			)
	}

	return check, nil
}

func ForgeItem(
	groupJID string,
	jid string,
	reference string,
	catalog *Catalog,
	materials *MaterialCatalog,
) (
	*ForgeResult,
	error,
) {
	if materials == nil {
		return nil,
			errors.New(
				"catálogo de materiais não inicializado",
			)
	}

	if err :=
		ValidateCraftRecipes(
			catalog,
			materials,
		); err != nil {

		return nil, err
	}

	recipe, err :=
		FindForgeRecipe(
			catalog,
			reference,
		)

	if err != nil {
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
				"erro iniciando forja: %w",
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

	if recipe.Item.Unique {
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
				recipe.Item.ID,
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
					"erro consultando item único na forja: %w",
					err,
				)
		}
	}

	var gold int

	err =
		tx.QueryRow(`
			SELECT gold
			FROM group_wallets
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&gold,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			ErrWalletNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando Gold da forja: %w",
				err,
			)
	}

	if gold <
		recipe.GoldCost {

		return nil,
			&ForgeGoldError{
				Required: recipe.GoldCost,

				Owned: gold,
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

	for _, requirement := range recipe.Item.
		CraftMaterials {

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
					"erro consultando material %s na forja: %w",
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
						MaterialID: requirement.
							MaterialID,

						Required: requirement.
							Quantity,

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

	result, err :=
		tx.Exec(`
			UPDATE group_wallets
			SET
				gold = gold - ?,
				updated_at =
					CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
			  AND gold >= ?
		`,
			recipe.GoldCost,
			groupJID,
			jid,
			recipe.GoldCost,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro debitando Gold da forja: %w",
				err,
			)
	}

	rowsAffected, err :=
		result.RowsAffected()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro validando débito da forja: %w",
				err,
			)
	}

	if rowsAffected != 1 {
		return nil,
			&ForgeGoldError{
				Required: recipe.GoldCost,

				Owned: gold,
			}
	}

	description :=
		fmt.Sprintf(
			"Forja %s (%s)",
			recipe.Item.Name,
			recipe.Code,
		)

	_, err =
		tx.Exec(`
			INSERT INTO gold_transactions (
				group_jid,
				jid,
				related_jid,
				amount,
				type,
				description
			)
			VALUES (?, ?, NULL, ?, ?, ?)
		`,
			groupJID,
			jid,
			-recipe.GoldCost,
			"RPG_FORGE",
			description,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro registrando custo Gold da forja: %w",
				err,
			)
	}

	for _, requirement := range recipe.Item.
		CraftMaterials {

		owned := ownedQuantities[requirement.MaterialID]

		remaining :=
			owned -
				requirement.
					Quantity

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
					requirement.
						MaterialID,
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
					requirement.
						MaterialID,
				)
		}

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro consumindo material %s na forja: %w",
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
			recipe.Item.ID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro adicionando equipamento forjado: %w",
				err,
			)
	}

	newGoldBalance :=
		gold -
			recipe.GoldCost

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando forja: %w",
				err,
			)
	}

	return &ForgeResult{
		Recipe: recipe,

		Quantity: 1,

		GoldBalance: newGoldBalance,
	}, nil
}
