package rpg

type Currency string

const (
	CurrencyGold         Currency = "GOLD"
	CurrencyMagicCrystal Currency = "MAGIC_CRYSTAL"
	CurrencyStellarStone Currency = "STELLAR_STONE"
)

func (currency Currency) Valid() bool {
	switch currency {
	case CurrencyGold,
		CurrencyMagicCrystal,
		CurrencyStellarStone:

		return true
	}

	return false
}

type Rarity string

const (
	RarityWorn      Rarity = "WORN"
	RarityCommon    Rarity = "COMMON"
	RarityRare      Rarity = "RARE"
	RarityEpic      Rarity = "EPIC"
	RarityLegendary Rarity = "LEGENDARY"
	RarityMythic    Rarity = "MYTHIC"
	RaritySacred    Rarity = "SACRED"
)

func (rarity Rarity) Valid() bool {
	switch rarity {
	case RarityWorn,
		RarityCommon,
		RarityRare,
		RarityEpic,
		RarityLegendary,
		RarityMythic,
		RaritySacred:

		return true
	}

	return false
}

func (rarity Rarity) Rank() int {
	switch rarity {
	case RarityWorn:
		return 1

	case RarityCommon:
		return 2

	case RarityRare:
		return 3

	case RarityEpic:
		return 4

	case RarityLegendary:
		return 5

	case RarityMythic:
		return 6

	case RaritySacred:
		return 7
	}

	return 0
}

type ItemType string

const (
	ItemTypeWeapon ItemType = "WEAPON"
	ItemTypeShield ItemType = "SHIELD"
	ItemTypeArmor  ItemType = "ARMOR"
)

func (itemType ItemType) Valid() bool {
	switch itemType {
	case ItemTypeWeapon,
		ItemTypeShield,
		ItemTypeArmor:

		return true
	}

	return false
}

type EquipmentSlot string

const (
	SlotWeapon EquipmentSlot = "weapon"
	SlotShield EquipmentSlot = "shield"
	SlotArmor  EquipmentSlot = "armor"
)

func (slot EquipmentSlot) Valid() bool {
	switch slot {
	case SlotWeapon,
		SlotShield,
		SlotArmor:

		return true
	}

	return false
}

type SourceType string

const (
	SourceCraft              SourceType = "CRAFT"
	SourceMonsterDrop        SourceType = "MONSTER_DROP"
	SourceBossDrop           SourceType = "BOSS_DROP"
	SourceShop               SourceType = "SHOP"
	SourceArcaneShop         SourceType = "ARCANE_SHOP"
	SourceOtherworldMerchant SourceType = "OTHERWORLD_MERCHANT"
)

func (source SourceType) Valid() bool {
	switch source {
	case SourceCraft,
		SourceMonsterDrop,
		SourceBossDrop,
		SourceShop,
		SourceArcaneShop,
		SourceOtherworldMerchant:

		return true
	}

	return false
}

type BonusType string

const (
	BonusCombatPowerPercent BonusType = "COMBAT_POWER_PERCENT"

	BonusDropChancePercent BonusType = "DROP_CHANCE_PERCENT"

	BonusRareDropChancePercent BonusType = "RARE_DROP_CHANCE_PERCENT"

	BonusGameLuckPercent BonusType = "GAME_LUCK_PERCENT"

	BonusCrystalRewardPercent BonusType = "CRYSTAL_REWARD_PERCENT"

	BonusStellarStoneChancePercent BonusType = "STELLAR_STONE_CHANCE_PERCENT"

	BonusDungeonSuccessPercent BonusType = "DUNGEON_SUCCESS_PERCENT"

	BonusBossDamagePercent BonusType = "BOSS_DAMAGE_PERCENT"

	BonusBlessingChancePercent BonusType = "BLESSING_CHANCE_PERCENT"

	BonusDropRarityUpgradeChancePercent BonusType = "DROP_RARITY_UPGRADE_CHANCE_PERCENT"
)

func (bonus BonusType) Valid() bool {
	switch bonus {
	case BonusCombatPowerPercent,
		BonusDropChancePercent,
		BonusRareDropChancePercent,
		BonusGameLuckPercent,
		BonusCrystalRewardPercent,
		BonusStellarStoneChancePercent,
		BonusDungeonSuccessPercent,
		BonusBossDamagePercent,
		BonusBlessingChancePercent,
		BonusDropRarityUpgradeChancePercent:

		return true
	}

	return false
}

type MaterialRequirement struct {
	MaterialID string `json:"material_id"`

	Quantity int `json:"quantity"`
}

type Item struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Type ItemType `json:"type"`

	Rarity Rarity `json:"rarity"`

	Power int `json:"power"`

	Attack int `json:"attack"`

	Defense int `json:"defense"`

	Description string `json:"description"`

	Origin string `json:"origin"`

	Value int `json:"value"`

	Currency Currency `json:"currency"`

	Source SourceType `json:"source"`

	MonsterID string `json:"monster_id,omitempty"`

	BossID string `json:"boss_id,omitempty"`

	SetID string `json:"set_id,omitempty"`

	Unique bool `json:"unique,omitempty"`

	CraftMaterials []MaterialRequirement `json:"craft_materials,omitempty"`
}

type SetEffect struct {
	Type BonusType `json:"type"`

	Value int `json:"value"`

	Description string `json:"description"`
}

type SetBonus struct {
	RequiredPieces int `json:"required_pieces"`

	Effects []SetEffect `json:"effects"`
}

type ItemSet struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Rarity Rarity `json:"rarity"`

	Theme string `json:"theme"`

	Pieces []string `json:"pieces"`

	Bonuses []SetBonus `json:"bonuses"`
}

type ActiveSetBonus struct {
	SetID string

	SetName string

	EquippedPieces int

	RequiredPieces int

	Effects []SetEffect
}

type PlayerState struct {
	GroupJID string

	JID string

	MagicCrystals int

	StellarStones int
}

type InventoryEntry struct {
	ItemID string

	Quantity int
}

type EquippedItem struct {
	Slot EquipmentSlot

	ItemID string
}
