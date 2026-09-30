package rpg

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed data/items/*.json data/sets.json
var catalogFS embed.FS

type Catalog struct {
	items map[string]Item

	sets map[string]ItemSet

	orderedItems []Item

	orderedSets []ItemSet
}

func LoadCatalog() (*Catalog, error) {
	catalog := &Catalog{
		items: make(
			map[string]Item,
		),

		sets: make(
			map[string]ItemSet,
		),

		orderedItems: make(
			[]Item,
			0,
		),

		orderedSets: make(
			[]ItemSet,
			0,
		),
	}

	itemFiles, err :=
		fs.Glob(
			catalogFS,
			"data/items/*.json",
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro listando catálogos de itens: %w",
				err,
			)
	}

	sort.Strings(
		itemFiles,
	)

	for _, filename := range itemFiles {

		if err :=
			catalog.loadItemsFile(
				filename,
			); err != nil {

			return nil, err
		}
	}

	if err :=
		catalog.loadSetsFile(
			"data/sets.json",
		); err != nil {

		return nil, err
	}

	if err :=
		catalog.validateRelations(); err != nil {

		return nil, err
	}

	return catalog, nil
}

func (
	catalog *Catalog,
) loadItemsFile(
	filename string,
) error {
	data, err :=
		catalogFS.ReadFile(
			filename,
		)

	if err != nil {
		return fmt.Errorf(
			"erro lendo %s: %w",
			filename,
			err,
		)
	}

	var items []Item

	if err :=
		json.Unmarshal(
			data,
			&items,
		); err != nil {

		return fmt.Errorf(
			"JSON inválido em %s: %w",
			filename,
			err,
		)
	}

	for index, item := range items {

		if err :=
			validateCatalogItem(
				item,
			); err != nil {

			return fmt.Errorf(
				"%s item %d: %w",
				filename,
				index+1,
				err,
			)
		}

		if _, exists :=
			catalog.items[item.ID]; exists {

			return fmt.Errorf(
				"item duplicado no catálogo: %s",
				item.ID,
			)
		}

		catalog.items[item.ID] = item

		catalog.orderedItems =
			append(
				catalog.orderedItems,
				item,
			)
	}

	return nil
}

func (
	catalog *Catalog,
) loadSetsFile(
	filename string,
) error {
	data, err :=
		catalogFS.ReadFile(
			filename,
		)

	if err != nil {
		return fmt.Errorf(
			"erro lendo %s: %w",
			filename,
			err,
		)
	}

	var sets []ItemSet

	if err :=
		json.Unmarshal(
			data,
			&sets,
		); err != nil {

		return fmt.Errorf(
			"JSON inválido em %s: %w",
			filename,
			err,
		)
	}

	for index, set := range sets {

		if err :=
			validateCatalogSet(
				set,
			); err != nil {

			return fmt.Errorf(
				"%s set %d: %w",
				filename,
				index+1,
				err,
			)
		}

		if _, exists :=
			catalog.sets[set.ID]; exists {

			return fmt.Errorf(
				"set duplicado no catálogo: %s",
				set.ID,
			)
		}

		catalog.sets[set.ID] = set

		catalog.orderedSets =
			append(
				catalog.orderedSets,
				set,
			)
	}

	return nil
}

func (
	catalog *Catalog,
) validateRelations() error {
	for _, item := range catalog.orderedItems {

		if item.SetID == "" {
			continue
		}

		set, exists :=
			catalog.sets[item.SetID]

		if !exists {
			return fmt.Errorf(
				"item %s referencia set inexistente %s",
				item.ID,
				item.SetID,
			)
		}

		if !containsString(
			set.Pieces,
			item.ID,
		) {
			return fmt.Errorf(
				"item %s informa set %s, mas não aparece nas peças do set",
				item.ID,
				item.SetID,
			)
		}
	}

	for _, set := range catalog.orderedSets {

		for _, itemID := range set.Pieces {

			item, exists :=
				catalog.items[itemID]

			if !exists {
				return fmt.Errorf(
					"set %s referencia item inexistente %s",
					set.ID,
					itemID,
				)
			}

			if item.SetID !=
				set.ID {

				return fmt.Errorf(
					"item %s deveria apontar para o set %s",
					item.ID,
					set.ID,
				)
			}
		}
	}

	return nil
}

func validateCatalogItem(
	item Item,
) error {
	if strings.TrimSpace(
		item.ID,
	) == "" {

		return fmt.Errorf(
			"id vazio",
		)
	}

	if strings.TrimSpace(
		item.Name,
	) == "" {

		return fmt.Errorf(
			"nome vazio",
		)
	}

	if !item.Type.Valid() {
		return fmt.Errorf(
			"tipo inválido: %s",
			item.Type,
		)
	}

	if !item.Rarity.Valid() {
		return fmt.Errorf(
			"raridade inválida: %s",
			item.Rarity,
		)
	}

	if !item.Source.Valid() {
		return fmt.Errorf(
			"origem inválida: %s",
			item.Source,
		)
	}

	if item.Power < 0 ||
		item.Attack < 0 ||
		item.Defense < 0 {

		return fmt.Errorf(
			"atributos negativos",
		)
	}

	if item.Value < 0 {
		return fmt.Errorf(
			"valor negativo",
		)
	}

	if !item.Currency.Valid() {
		return fmt.Errorf(
			"moeda inválida: %s",
			item.Currency,
		)
	}

	for _, material := range item.CraftMaterials {

		if strings.TrimSpace(
			material.MaterialID,
		) == "" {

			return fmt.Errorf(
				"material de crafting sem id",
			)
		}

		if material.Quantity <= 0 {
			return fmt.Errorf(
				"quantidade inválida para material %s",
				material.MaterialID,
			)
		}
	}

	return nil
}

func validateCatalogSet(
	set ItemSet,
) error {
	if strings.TrimSpace(
		set.ID,
	) == "" {

		return fmt.Errorf(
			"id vazio",
		)
	}

	if strings.TrimSpace(
		set.Name,
	) == "" {

		return fmt.Errorf(
			"nome vazio",
		)
	}

	if !set.Rarity.Valid() {
		return fmt.Errorf(
			"raridade inválida: %s",
			set.Rarity,
		)
	}

	if len(set.Pieces) == 0 {
		return fmt.Errorf(
			"set sem peças",
		)
	}

	seen :=
		make(
			map[string]bool,
		)

	for _, piece := range set.Pieces {

		if seen[piece] {
			return fmt.Errorf(
				"peça duplicada: %s",
				piece,
			)
		}

		seen[piece] =
			true
	}

	for _, bonus := range set.Bonuses {

		if bonus.RequiredPieces <= 0 ||
			bonus.RequiredPieces >
				len(set.Pieces) {

			return fmt.Errorf(
				"quantidade inválida de peças para bônus",
			)
		}

		if len(
			bonus.Effects,
		) == 0 {

			return fmt.Errorf(
				"bônus sem efeitos",
			)
		}

		for _, effect := range bonus.Effects {

			if !effect.Type.Valid() {
				return fmt.Errorf(
					"tipo de bônus inválido: %s",
					effect.Type,
				)
			}

			if effect.Value <= 0 {
				return fmt.Errorf(
					"valor de bônus inválido",
				)
			}
		}
	}

	return nil
}

func (
	catalog *Catalog,
) ItemByID(
	itemID string,
) (Item, bool) {
	item, exists :=
		catalog.items[itemID]

	return item, exists
}

func (
	catalog *Catalog,
) SetByID(
	setID string,
) (ItemSet, bool) {
	set, exists :=
		catalog.sets[setID]

	return set, exists
}

func (
	catalog *Catalog,
) Items() []Item {
	result :=
		make(
			[]Item,
			len(
				catalog.orderedItems,
			),
		)

	copy(
		result,
		catalog.orderedItems,
	)

	return result
}

func (
	catalog *Catalog,
) Sets() []ItemSet {
	result :=
		make(
			[]ItemSet,
			len(
				catalog.orderedSets,
			),
		)

	copy(
		result,
		catalog.orderedSets,
	)

	return result
}

func (
	catalog *Catalog,
) ItemsByRarity(
	rarity Rarity,
) []Item {
	result :=
		make(
			[]Item,
			0,
		)

	for _, item := range catalog.orderedItems {

		if item.Rarity ==
			rarity {

			result =
				append(
					result,
					item,
				)
		}
	}

	return result
}

func (
	catalog *Catalog,
) ItemsBySource(
	source SourceType,
) []Item {
	result :=
		make(
			[]Item,
			0,
		)

	for _, item := range catalog.orderedItems {

		if item.Source ==
			source {

			result =
				append(
					result,
					item,
				)
		}
	}

	return result
}

func (
	catalog *Catalog,
) ActiveSetBonuses(
	equippedItemIDs []string,
) []ActiveSetBonus {
	setCounts :=
		make(
			map[string]int,
		)

	for _, itemID := range equippedItemIDs {

		item, exists :=
			catalog.items[itemID]

		if !exists ||
			item.SetID == "" {

			continue
		}

		setCounts[item.SetID]++
	}

	result :=
		make(
			[]ActiveSetBonus,
			0,
		)

	for _, set := range catalog.orderedSets {

		equipped :=
			setCounts[set.ID]

		if equipped == 0 {
			continue
		}

		for _, bonus := range set.Bonuses {

			if equipped <
				bonus.RequiredPieces {

				continue
			}

			result =
				append(
					result,
					ActiveSetBonus{
						SetID: set.ID,

						SetName: set.Name,

						EquippedPieces: equipped,

						RequiredPieces: bonus.RequiredPieces,

						Effects: bonus.Effects,
					},
				)
		}
	}

	return result
}

func containsString(
	values []string,
	target string,
) bool {
	for _, value := range values {

		if value == target {
			return true
		}
	}

	return false
}
