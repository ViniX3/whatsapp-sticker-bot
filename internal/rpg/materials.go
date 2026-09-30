package rpg

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type MaterialKind string

const (
	MaterialKindWood MaterialKind = "WOOD"

	MaterialKindMineral MaterialKind = "MINERAL"

	MaterialKindMetal MaterialKind = "METAL"

	MaterialKindGem MaterialKind = "GEM"

	MaterialKindHide MaterialKind = "HIDE"

	MaterialKindTextile MaterialKind = "TEXTILE"

	MaterialKindMonsterPart MaterialKind = "MONSTER_PART"

	MaterialKindBossPart MaterialKind = "BOSS_PART"

	MaterialKindOtherworld MaterialKind = "OTHERWORLD"
)

func (kind MaterialKind) Valid() bool {
	switch kind {
	case MaterialKindWood,
		MaterialKindMineral,
		MaterialKindMetal,
		MaterialKindGem,
		MaterialKindHide,
		MaterialKindTextile,
		MaterialKindMonsterPart,
		MaterialKindBossPart,
		MaterialKindOtherworld:

		return true
	}

	return false
}

type MaterialSource string

const (
	MaterialSourceGathering MaterialSource = "GATHERING"

	MaterialSourceMonsterDrop MaterialSource = "MONSTER_DROP"

	MaterialSourceBossDrop MaterialSource = "BOSS_DROP"

	MaterialSourceOtherworldMerchant MaterialSource = "OTHERWORLD_MERCHANT"
)

func (source MaterialSource) Valid() bool {
	switch source {
	case MaterialSourceGathering,
		MaterialSourceMonsterDrop,
		MaterialSourceBossDrop,
		MaterialSourceOtherworldMerchant:

		return true
	}

	return false
}

type Material struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Kind MaterialKind `json:"kind"`

	Rarity Rarity `json:"rarity"`

	Source MaterialSource `json:"source"`

	Value int `json:"value"`

	Description string `json:"description"`

	MonsterID string `json:"monster_id,omitempty"`

	BossID string `json:"boss_id,omitempty"`
}

//go:embed data/materials.json
var materialFS embed.FS

type MaterialCatalog struct {
	materials map[string]Material

	ordered []Material
}

func LoadMaterialCatalog() (*MaterialCatalog, error) {
	data, err :=
		materialFS.ReadFile(
			"data/materials.json",
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro lendo catálogo de materiais: %w",
				err,
			)
	}

	var materials []Material

	if err :=
		json.Unmarshal(
			data,
			&materials,
		); err != nil {

		return nil,
			fmt.Errorf(
				"JSON inválido no catálogo de materiais: %w",
				err,
			)
	}

	catalog :=
		&MaterialCatalog{
			materials: make(
				map[string]Material,
			),

			ordered: make(
				[]Material,
				0,
				len(materials),
			),
		}

	for index, material := range materials {

		if err :=
			validateMaterial(
				material,
			); err != nil {

			return nil,
				fmt.Errorf(
					"material %d: %w",
					index+1,
					err,
				)
		}

		if _, exists :=
			catalog.materials[material.ID]; exists {

			return nil,
				fmt.Errorf(
					"material duplicado: %s",
					material.ID,
				)
		}

		catalog.materials[material.ID] =
			material

		catalog.ordered =
			append(
				catalog.ordered,
				material,
			)
	}

	sort.Slice(
		catalog.ordered,
		func(
			i int,
			j int,
		) bool {
			return catalog.ordered[i].ID <
				catalog.ordered[j].ID
		},
	)

	return catalog, nil
}

func validateMaterial(
	material Material,
) error {
	if strings.TrimSpace(
		material.ID,
	) == "" {

		return fmt.Errorf(
			"id vazio",
		)
	}

	if strings.TrimSpace(
		material.Name,
	) == "" {

		return fmt.Errorf(
			"nome vazio no material %s",
			material.ID,
		)
	}

	if !material.Kind.Valid() {
		return fmt.Errorf(
			"tipo inválido no material %s: %s",
			material.ID,
			material.Kind,
		)
	}

	if !material.Rarity.Valid() {
		return fmt.Errorf(
			"raridade inválida no material %s: %s",
			material.ID,
			material.Rarity,
		)
	}

	if !material.Source.Valid() {
		return fmt.Errorf(
			"origem inválida no material %s: %s",
			material.ID,
			material.Source,
		)
	}

	if material.Value < 0 {
		return fmt.Errorf(
			"valor negativo no material %s",
			material.ID,
		)
	}

	switch material.Source {

	case MaterialSourceMonsterDrop:
		if strings.TrimSpace(
			material.MonsterID,
		) == "" {

			return fmt.Errorf(
				"material %s é drop de monstro mas não possui monster_id",
				material.ID,
			)
		}

	case MaterialSourceBossDrop:
		if strings.TrimSpace(
			material.BossID,
		) == "" {

			return fmt.Errorf(
				"material %s é drop de boss mas não possui boss_id",
				material.ID,
			)
		}
	}

	return nil
}

func (
	catalog *MaterialCatalog,
) MaterialByID(
	materialID string,
) (Material, bool) {
	material, exists :=
		catalog.materials[materialID]

	return material, exists
}

func (
	catalog *MaterialCatalog,
) Materials() []Material {
	result :=
		make(
			[]Material,
			len(catalog.ordered),
		)

	copy(
		result,
		catalog.ordered,
	)

	return result
}

func (
	catalog *MaterialCatalog,
) MaterialsByKind(
	kind MaterialKind,
) []Material {
	result :=
		make(
			[]Material,
			0,
		)

	for _, material := range catalog.ordered {

		if material.Kind != kind {
			continue
		}

		result =
			append(
				result,
				material,
			)
	}

	return result
}

func (
	catalog *MaterialCatalog,
) MaterialsBySource(
	source MaterialSource,
) []Material {
	result :=
		make(
			[]Material,
			0,
		)

	for _, material := range catalog.ordered {

		if material.Source != source {
			continue
		}

		result =
			append(
				result,
				material,
			)
	}

	return result
}
