package rpg

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

const (
	GatheringCooldown = 2 * time.Minute

	GatheringBaseRolls = 3

	GatheringBonusRollChancePercent = 5
)

var (
	ErrGatheringCooldown = errors.New(
		"coleta em cooldown",
	)

	ErrInvalidGatheringRegion = errors.New(
		"região de coleta inválida",
	)

	ErrNoGatheringMaterials = errors.New(
		"nenhum material disponível para coleta",
	)
)

type GatheringRegion string

const (
	GatheringForest GatheringRegion = "FOREST"

	GatheringQuarry GatheringRegion = "QUARRY"

	GatheringMine GatheringRegion = "MINE"
)

func (region GatheringRegion) Valid() bool {
	switch region {
	case GatheringForest,
		GatheringQuarry,
		GatheringMine:
		return true
	}

	return false
}

func (region GatheringRegion) Name() string {
	switch region {
	case GatheringForest:
		return "Floresta"

	case GatheringQuarry:
		return "Pedreira"

	case GatheringMine:
		return "Mina"
	}

	return "Região Desconhecida"
}

func (region GatheringRegion) Icon() string {
	switch region {
	case GatheringForest:
		return "🌲"

	case GatheringQuarry:
		return "🪨"

	case GatheringMine:
		return "⛏️"
	}

	return "🗺️"
}

type GatheringDrop struct {
	Material Material

	Quantity int
}

type GatheringResult struct {
	Region GatheringRegion

	Drops []GatheringDrop

	Rolls int

	BonusRoll bool

	TotalItems int

	EpicItems int

	LegendaryItems int

	GatheredAt time.Time

	NextAvailableAt time.Time
}

type GatheringStats struct {
	TotalGatherings int

	TotalResources int

	EpicResources int

	LegendaryResources int

	LastGatheredAt *time.Time
}

type GatheringCooldownError struct {
	Region GatheringRegion

	Remaining time.Duration

	NextAvailableAt time.Time
}

func (err *GatheringCooldownError) Error() string {
	return fmt.Sprintf(
		"%s: %s restantes",
		ErrGatheringCooldown,
		err.Remaining.Round(time.Second),
	)
}

func (err *GatheringCooldownError) Unwrap() error {
	return ErrGatheringCooldown
}

// Os pools são explícitos de propósito.
//
// Não usamos automaticamente todos os materiais cujo source seja GATHERING,
// pois alguns recursos, como lingotes, devem ser obtidos futuramente através
// da Forja e nunca diretamente pelo !coletar.
var gatheringPools = map[GatheringRegion]map[Rarity][]string{
	GatheringForest: {
		RarityWorn: {
			"wood_branch",
		},

		RarityCommon: {
			"wood_pine",
		},

		RarityRare: {
			"wood_oak",
		},

		RarityEpic: {
			"wood_ancient",
		},
	},

	GatheringQuarry: {
		RarityWorn: {
			"stone",
			"clay",
		},

		RarityCommon: {
			"granite",
		},

		RarityRare: {
			"quartz",
		},

		RarityEpic: {
			"obsidian",
		},
	},

	GatheringMine: {
		RarityCommon: {
			"coal",
			"ore_copper",
			"ore_tin",
			"ore_iron",
		},

		RarityRare: {
			"ore_silver",
			"ore_gold",
		},

		RarityEpic: {
			"ore_mithril",
			"sapphire",
			"ruby",
			"emerald",
		},

		RarityLegendary: {
			"diamond",
			"ore_adamantite",
		},
	},
}

var gatheringRarityOrder = []Rarity{
	RarityWorn,
	RarityCommon,
	RarityRare,
	RarityEpic,
	RarityLegendary,
}

func InitGathering() error {
	if err := ensureSchema(); err != nil {
		return err
	}

	return ensureGatheringSchema()
}

func ParseGatheringRegion(
	value string,
) (GatheringRegion, bool) {
	switch strings.ToLower(
		strings.TrimSpace(value),
	) {
	case "floresta",
		"forest":
		return GatheringForest, true

	case "pedreira",
		"quarry":
		return GatheringQuarry, true

	case "mina",
		"mine":
		return GatheringMine, true
	}

	return "", false
}

func ValidateGatheringPools(
	catalog *MaterialCatalog,
) error {
	if catalog == nil {
		return errors.New(
			"catálogo de materiais não inicializado",
		)
	}

	for region, rarityPools := range gatheringPools {
		if !region.Valid() {
			return fmt.Errorf(
				"%w: %s",
				ErrInvalidGatheringRegion,
				region,
			)
		}

		if len(rarityPools) == 0 {
			return fmt.Errorf(
				"%w na região %s",
				ErrNoGatheringMaterials,
				region,
			)
		}

		for rarity, materialIDs := range rarityPools {
			if !rarity.Valid() {
				return fmt.Errorf(
					"raridade inválida no pool de %s: %s",
					region,
					rarity,
				)
			}

			if len(materialIDs) == 0 {
				return fmt.Errorf(
					"pool vazio em %s / %s",
					region,
					rarity,
				)
			}

			seen := make(map[string]bool)

			for _, materialID := range materialIDs {
				if seen[materialID] {
					return fmt.Errorf(
						"material duplicado em %s / %s: %s",
						region,
						rarity,
						materialID,
					)
				}

				seen[materialID] = true

				material, exists :=
					catalog.MaterialByID(
						materialID,
					)

				if !exists {
					return fmt.Errorf(
						"material inexistente no pool de coleta: %s",
						materialID,
					)
				}

				if material.Source !=
					MaterialSourceGathering {

					return fmt.Errorf(
						"material %s não possui origem GATHERING",
						materialID,
					)
				}

				if material.Rarity != rarity {
					return fmt.Errorf(
						"material %s possui raridade %s, mas está no pool %s",
						materialID,
						material.Rarity,
						rarity,
					)
				}
			}
		}
	}

	return nil
}

func Gather(
	groupJID string,
	jid string,
	region GatheringRegion,
	catalog *MaterialCatalog,
) (*GatheringResult, error) {
	if catalog == nil {
		return nil,
			errors.New(
				"catálogo de materiais não inicializado",
			)
	}

	if err :=
		ValidateGatheringPools(
			catalog,
		); err != nil {

		return nil, err
	}

	if region == "" {
		randomRegion, err :=
			randomGatheringRegion()

		if err != nil {
			return nil, err
		}

		region =
			randomRegion
	}

	if !region.Valid() {
		return nil,
			ErrInvalidGatheringRegion
	}

	if err := ensureSchema(); err != nil {
		return nil, err
	}

	if err :=
		ensureGatheringSchema(); err != nil {

		return nil, err
	}

	if err :=
		ensureGatheringRegionSchema(); err != nil {

		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando coleta: %w",
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

	_, err =
		tx.Exec(`
			INSERT INTO rpg_gathering (
				group_jid,
				jid
			)
			VALUES (?, ?)

			ON CONFLICT (
				group_jid,
				jid
			)
			DO NOTHING
		`,
			groupJID,
			jid,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro criando estado de coleta: %w",
				err,
			)
	}

	var lastGathered sql.NullTime

	err =
		tx.QueryRow(`
			SELECT last_gathered_at
			FROM rpg_gathering_regions
			WHERE group_jid = ?
			  AND jid = ?
			  AND region = ?
		`,
			groupJID,
			jid,
			string(region),
		).Scan(
			&lastGathered,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		lastGathered =
			sql.NullTime{}

	} else if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando cooldown regional de coleta: %w",
				err,
			)
	}

	now :=
		time.Now().
			UTC().
			Truncate(time.Second)

	if lastGathered.Valid {
		nextAvailable :=
			lastGathered.Time.
				Add(GatheringCooldown)

		if now.Before(
			nextAvailable,
		) {
			remaining :=
				nextAvailable.
					Sub(now)

			return nil,
				&GatheringCooldownError{
					Region: region,

					Remaining: remaining,

					NextAvailableAt: nextAvailable,
				}
		}
	}

	bonusRoll, err :=
		gatheringChance(
			GatheringBonusRollChancePercent,
		)

	if err != nil {
		return nil, err
	}

	rolls :=
		GatheringBaseRolls

	if bonusRoll {
		rolls++
	}

	drops :=
		make(
			[]GatheringDrop,
			0,
			rolls,
		)

	dropIndex :=
		make(
			map[string]int,
		)

	totalItems := 0

	epicItems := 0

	legendaryItems := 0

	for roll := 0; roll < rolls; roll++ {
		rolledRarity, err :=
			rollGatheringRarity()

		if err != nil {
			return nil, err
		}

		actualRarity, err :=
			resolveGatheringRarity(
				region,
				rolledRarity,
			)

		if err != nil {
			return nil, err
		}

		material, err :=
			randomGatheringMaterial(
				region,
				actualRarity,
				catalog,
			)

		if err != nil {
			return nil, err
		}

		quantity, err :=
			randomGatheringQuantity(
				material.Rarity,
			)

		if err != nil {
			return nil, err
		}

		if index, exists :=
			dropIndex[material.ID]; exists {

			drops[index].Quantity +=
				quantity
		} else {
			dropIndex[material.ID] =
				len(drops)

			drops =
				append(
					drops,
					GatheringDrop{
						Material: material,

						Quantity: quantity,
					},
				)
		}

		totalItems +=
			quantity

		switch material.Rarity {
		case RarityEpic:
			epicItems +=
				quantity

		case RarityLegendary:
			legendaryItems +=
				quantity
		}
	}

	if len(drops) == 0 {
		return nil,
			ErrNoGatheringMaterials
	}

	for _, drop := range drops {
		_, err =
			tx.Exec(`
				INSERT INTO rpg_inventory (
					group_jid,
					jid,
					item_id,
					quantity
				)
				VALUES (?, ?, ?, ?)

				ON CONFLICT (
					group_jid,
					jid,
					item_id
				)
				DO UPDATE SET
					quantity =
						rpg_inventory.quantity
						+ excluded.quantity,

					updated_at =
						CURRENT_TIMESTAMP
			`,
				groupJID,
				jid,
				drop.Material.ID,
				drop.Quantity,
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro adicionando material %s ao inventário: %w",
					drop.Material.ID,
					err,
				)
		}
	}

	_, err =
		tx.Exec(`
			UPDATE rpg_gathering
			SET
				last_gathered_at = ?,

				total_gatherings =
					total_gatherings + 1,

				total_resources =
					total_resources + ?,

				epic_resources =
					epic_resources + ?,

				legendary_resources =
					legendary_resources + ?,

				updated_at =
					CURRENT_TIMESTAMP

			WHERE group_jid = ?
			  AND jid = ?
		`,
			now,
			totalItems,
			epicItems,
			legendaryItems,
			groupJID,
			jid,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro atualizando estatísticas de coleta: %w",
				err,
			)
	}

	_, err =
		tx.Exec(`
			INSERT INTO rpg_gathering_regions (
				group_jid,
				jid,
				region,
				last_gathered_at
			)
			VALUES (?, ?, ?, ?)

			ON CONFLICT (
				group_jid,
				jid,
				region
			)
			DO UPDATE SET
				last_gathered_at =
					excluded.last_gathered_at,

				updated_at =
					CURRENT_TIMESTAMP
		`,
			groupJID,
			jid,
			string(region),
			now,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro atualizando cooldown regional: %w",
				err,
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando coleta: %w",
				err,
			)
	}

	return &GatheringResult{
		Region: region,

		Drops: drops,

		Rolls: rolls,

		BonusRoll: bonusRoll,

		TotalItems: totalItems,

		EpicItems: epicItems,

		LegendaryItems: legendaryItems,

		GatheredAt: now,

		NextAvailableAt: now.Add(
			GatheringCooldown,
		),
	}, nil
}

func GetGatheringStats(
	groupJID string,
	jid string,
) (*GatheringStats, error) {
	if err :=
		ensureGatheringSchema(); err != nil {

		return nil, err
	}

	stats :=
		&GatheringStats{}

	var lastGathered sql.NullTime

	err :=
		database.DB.QueryRow(`
			SELECT
				total_gatherings,
				total_resources,
				epic_resources,
				legendary_resources,
				last_gathered_at
			FROM rpg_gathering
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&stats.TotalGatherings,
			&stats.TotalResources,
			&stats.EpicResources,
			&stats.LegendaryResources,
			&lastGathered,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return stats, nil
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando estatísticas de coleta: %w",
				err,
			)
	}

	if lastGathered.Valid {
		value :=
			lastGathered.Time

		stats.LastGatheredAt =
			&value
	}

	return stats, nil
}

func GetGatheringRemainingCooldown(
	groupJID string,
	jid string,
) (time.Duration, error) {
	if err :=
		ensureGatheringSchema(); err != nil {

		return 0, err
	}

	var lastGathered sql.NullTime

	err :=
		database.DB.QueryRow(`
			SELECT last_gathered_at
			FROM rpg_gathering
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&lastGathered,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return 0, nil
	}

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro consultando cooldown de coleta: %w",
				err,
			)
	}

	if !lastGathered.Valid {
		return 0, nil
	}

	next :=
		lastGathered.Time.
			Add(GatheringCooldown)

	now :=
		time.Now().
			UTC()

	if !now.Before(next) {
		return 0, nil
	}

	return next.Sub(now), nil
}

// GetGatheringRegionRemainingCooldown retorna o tempo
// restante somente da região informada.
func GetGatheringRegionRemainingCooldown(
	groupJID string,
	jid string,
	region GatheringRegion,
) (time.Duration, error) {
	if !region.Valid() {
		return 0, ErrInvalidGatheringRegion
	}

	if err :=
		ensureGatheringRegionSchema(); err != nil {

		return 0, err
	}

	var lastGathered sql.NullTime

	err :=
		database.DB.QueryRow(`
			SELECT last_gathered_at
			FROM rpg_gathering_regions
			WHERE group_jid = ?
			  AND jid = ?
			  AND region = ?
		`,
			groupJID,
			jid,
			string(region),
		).Scan(
			&lastGathered,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return 0, nil
	}

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro consultando cooldown regional: %w",
				err,
			)
	}

	if !lastGathered.Valid {
		return 0, nil
	}

	next :=
		lastGathered.Time.
			Add(GatheringCooldown)

	now :=
		time.Now().
			UTC()

	if !now.Before(next) {
		return 0, nil
	}

	return next.Sub(now), nil
}

func ensureGatheringRegionSchema() error {
	if database.DB == nil {
		return errors.New(
			"banco de dados não inicializado",
		)
	}

	_, err :=
		database.DB.Exec(`
			CREATE TABLE IF NOT EXISTS rpg_gathering_regions (
				group_jid TEXT NOT NULL,
				jid TEXT NOT NULL,

				region TEXT NOT NULL
					CHECK (
						region IN (
							'FOREST',
							'QUARRY',
							'MINE'
						)
					),

				last_gathered_at DATETIME,

				created_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,

				updated_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,

				PRIMARY KEY (
					group_jid,
					jid,
					region
				),

				FOREIGN KEY (
					group_jid,
					jid
				)
					REFERENCES rpg_players (
						group_jid,
						jid
					)
					ON DELETE CASCADE
			);

			CREATE INDEX IF NOT EXISTS
				idx_rpg_gathering_regions_cooldown
			ON rpg_gathering_regions (
				group_jid,
				jid,
				region,
				last_gathered_at
			);
		`)

	if err != nil {
		return fmt.Errorf(
			"erro criando cooldown regional de coleta: %w",
			err,
		)
	}

	return nil
}

func ensureGatheringSchema() error {
	if database.DB == nil {
		return errors.New(
			"banco de dados não inicializado",
		)
	}

	_, err :=
		database.DB.Exec(`
			CREATE TABLE IF NOT EXISTS rpg_gathering (
				group_jid TEXT NOT NULL,
				jid TEXT NOT NULL,

				last_gathered_at DATETIME,

				total_gatherings INTEGER NOT NULL DEFAULT 0
					CHECK (total_gatherings >= 0),

				total_resources INTEGER NOT NULL DEFAULT 0
					CHECK (total_resources >= 0),

				epic_resources INTEGER NOT NULL DEFAULT 0
					CHECK (epic_resources >= 0),

				legendary_resources INTEGER NOT NULL DEFAULT 0
					CHECK (legendary_resources >= 0),

				created_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,

				updated_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,

				PRIMARY KEY (
					group_jid,
					jid
				),

				FOREIGN KEY (
					group_jid,
					jid
				)
					REFERENCES rpg_players (
						group_jid,
						jid
					)
					ON DELETE CASCADE
			);
		`)

	if err != nil {
		return fmt.Errorf(
			"erro criando tabela rpg_gathering: %w",
			err,
		)
	}

	return nil
}

func randomGatheringRegion() (
	GatheringRegion,
	error,
) {
	regions :=
		[]GatheringRegion{
			GatheringForest,
			GatheringQuarry,
			GatheringMine,
		}

	index, err :=
		gatheringRandomInt(
			len(regions),
		)

	if err != nil {
		return "",
			err
	}

	return regions[index],
		nil
}

func rollGatheringRarity() (
	Rarity,
	error,
) {
	roll, err :=
		gatheringRandomInt(
			10000,
		)

	if err != nil {
		return "",
			err
	}

	switch {
	case roll < 2500:
		return RarityWorn, nil

	case roll < 8500:
		return RarityCommon, nil

	case roll < 9700:
		return RarityRare, nil

	case roll < 9970:
		return RarityEpic, nil

	default:
		return RarityLegendary, nil
	}
}

func resolveGatheringRarity(
	region GatheringRegion,
	target Rarity,
) (Rarity, error) {
	pools, exists :=
		gatheringPools[region]

	if !exists {
		return "",
			ErrInvalidGatheringRegion
	}

	if materialIDs :=
		pools[target]; len(materialIDs) > 0 {

		return target, nil
	}

	targetIndex := -1

	for index, rarity := range gatheringRarityOrder {

		if rarity == target {
			targetIndex =
				index

			break
		}
	}

	if targetIndex == -1 {
		return "",
			fmt.Errorf(
				"raridade de coleta inválida: %s",
				target,
			)
	}

	// Primeiro tentamos uma raridade inferior.
	//
	// Exemplo:
	// Floresta não possui material Lendário.
	// Um roll Lendário será convertido em Épico.
	for index :=
		targetIndex - 1; index >= 0; index-- {

		rarity :=
			gatheringRarityOrder[index]

		if len(
			pools[rarity],
		) > 0 {

			return rarity,
				nil
		}
	}

	// Caso não exista nenhuma raridade inferior,
	// procuramos a próxima superior.
	//
	// Isso ocorre atualmente na Mina:
	// ela não possui materiais Desgastados,
	// portanto um roll Desgastado vira Comum.
	for index :=
		targetIndex + 1; index < len(gatheringRarityOrder); index++ {

		rarity :=
			gatheringRarityOrder[index]

		if len(
			pools[rarity],
		) > 0 {

			return rarity,
				nil
		}
	}

	return "",
		fmt.Errorf(
			"%w na região %s",
			ErrNoGatheringMaterials,
			region,
		)
}

func randomGatheringMaterial(
	region GatheringRegion,
	rarity Rarity,
	catalog *MaterialCatalog,
) (Material, error) {
	pools, exists :=
		gatheringPools[region]

	if !exists {
		return Material{},
			ErrInvalidGatheringRegion
	}

	materialIDs :=
		pools[rarity]

	if len(materialIDs) == 0 {
		return Material{},
			fmt.Errorf(
				"%w em %s / %s",
				ErrNoGatheringMaterials,
				region,
				rarity,
			)
	}

	index, err :=
		gatheringRandomInt(
			len(materialIDs),
		)

	if err != nil {
		return Material{},
			err
	}

	materialID :=
		materialIDs[index]

	material, exists :=
		catalog.MaterialByID(
			materialID,
		)

	if !exists {
		return Material{},
			fmt.Errorf(
				"material de coleta não encontrado: %s",
				materialID,
			)
	}

	return material, nil
}

func randomGatheringQuantity(
	rarity Rarity,
) (int, error) {
	switch rarity {
	case RarityWorn:
		return gatheringRandomRange(
			3,
			6,
		)

	case RarityCommon:
		return gatheringRandomRange(
			2,
			5,
		)

	case RarityRare:
		return gatheringRandomRange(
			1,
			3,
		)

	case RarityEpic:
		return gatheringRandomRange(
			1,
			2,
		)

	case RarityLegendary:
		return 1, nil
	}

	return 0,
		fmt.Errorf(
			"raridade inválida para quantidade de coleta: %s",
			rarity,
		)
}

func gatheringChance(
	percent int,
) (bool, error) {
	if percent <= 0 {
		return false, nil
	}

	if percent >= 100 {
		return true, nil
	}

	value, err :=
		gatheringRandomInt(
			100,
		)

	if err != nil {
		return false, err
	}

	return value < percent,
		nil
}

func gatheringRandomRange(
	minimum int,
	maximum int,
) (int, error) {
	if minimum > maximum {
		return 0,
			fmt.Errorf(
				"intervalo aleatório inválido: %d-%d",
				minimum,
				maximum,
			)
	}

	size :=
		maximum -
			minimum +
			1

	value, err :=
		gatheringRandomInt(
			size,
		)

	if err != nil {
		return 0, err
	}

	return minimum + value,
		nil
}

func gatheringRandomInt(
	maximum int,
) (int, error) {
	if maximum <= 0 {
		return 0,
			fmt.Errorf(
				"limite aleatório inválido: %d",
				maximum,
			)
	}

	value, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(
				int64(maximum),
			),
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro gerando aleatoriedade segura: %w",
				err,
			)
	}

	return int(
		value.Int64(),
	), nil
}
