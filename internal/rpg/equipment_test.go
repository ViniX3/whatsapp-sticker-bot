package rpg

import "testing"

func TestEquipmentSlotMapping(
	t *testing.T,
) {
	tests :=
		[]struct {
			ItemType ItemType
			Expected EquipmentSlot
		}{
			{
				ItemType: ItemTypeWeapon,

				Expected: SlotWeapon,
			},
			{
				ItemType: ItemTypeShield,

				Expected: SlotShield,
			},
			{
				ItemType: ItemTypeArmor,

				Expected: SlotArmor,
			},
		}

	for _, test := range tests {

		slot, valid :=
			equipmentSlotForItemType(
				test.ItemType,
			)

		if !valid {
			t.Fatalf(
				"tipo %s deveria possuir slot",
				test.ItemType,
			)
		}

		if slot !=
			test.Expected {

			t.Fatalf(
				"tipo %s: esperado slot %s, recebido %s",
				test.ItemType,
				test.Expected,
				slot,
			)
		}
	}
}

func TestEquipmentSlotsAreValid(
	t *testing.T,
) {
	slots :=
		[]EquipmentSlot{
			SlotWeapon,
			SlotShield,
			SlotArmor,
		}

	for _, slot := range slots {

		if !slot.Valid() {
			t.Fatalf(
				"slot deveria ser válido: %s",
				slot,
			)
		}
	}
}

func TestMythicTitanSetBonuses(
	t *testing.T,
) {
	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	active :=
		catalog.ActiveSetBonuses(
			[]string{
				"mythic_ancient_titan_weapon",
				"mythic_ancient_titan_shield",
				"mythic_ancient_titan_armor",
			},
		)

	if len(active) != 2 {
		t.Fatalf(
			"esperados 2 bônus ativos, encontrados %d",
			len(active),
		)
	}

	combatPowerBonus :=
		0

	bossDamageBonus :=
		0

	for _, bonus := range active {

		for _, effect := range bonus.Effects {

			switch effect.Type {

			case BonusCombatPowerPercent:
				combatPowerBonus +=
					effect.Value

			case BonusBossDamagePercent:
				bossDamageBonus +=
					effect.Value
			}
		}
	}

	if combatPowerBonus != 10 {
		t.Fatalf(
			"esperado +10%% de Poder de Combate, encontrado %d%%",
			combatPowerBonus,
		)
	}

	if bossDamageBonus != 12 {
		t.Fatalf(
			"esperado +12%% contra Bosses, encontrado %d%%",
			bossDamageBonus,
		)
	}
}

func TestPartialSetBonus(
	t *testing.T,
) {
	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	active :=
		catalog.ActiveSetBonuses(
			[]string{
				"mythic_ancient_titan_weapon",
				"mythic_ancient_titan_shield",
			},
		)

	if len(active) != 1 {
		t.Fatalf(
			"esperado apenas bônus de 2 peças, encontrados %d",
			len(active),
		)
	}

	if active[0].RequiredPieces != 2 {
		t.Fatalf(
			"esperado bônus de 2 peças, encontrado %d",
			active[0].RequiredPieces,
		)
	}
}
