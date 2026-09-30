package rpg

import (
	"errors"
	"testing"
	"time"
)

func TestRaidParticipantFromEquipmentSummary(
	t *testing.T,
) {
	summary :=
		&EquipmentSummary{
			CombatPower: 10000,

			ActiveSetBonuses: []ActiveSetBonus{
				{
					SetID: "test-set",

					SetName: "Test Set",

					EquippedPieces: 3,

					RequiredPieces: 3,

					Effects: []SetEffect{
						{
							Type: BonusBossDamagePercent,

							Value: 20,
						},
						{
							Type: BonusCrystalRewardPercent,

							Value: 15,
						},
						{
							Type: BonusStellarStoneChancePercent,

							Value: 4,
						},
						{
							Type: BonusBlessingChancePercent,

							Value: 3,
						},
						{
							Type: BonusDropChancePercent,

							Value: 8,
						},
						{
							Type: BonusRareDropChancePercent,

							Value: 5,
						},
						{
							Type: BonusDropRarityUpgradeChancePercent,

							Value: 2,
						},
					},
				},
			},
		}

	participant, err :=
		RaidParticipantFromEquipmentSummary(
			"player-a",
			"Player A",
			summary,
		)

	if err != nil {
		t.Fatal(err)
	}

	if participant.CombatPower != 10000 {
		t.Fatalf(
			"PC esperado 10000, recebido %d",
			participant.CombatPower,
		)
	}

	if participant.EffectivePower != 12000 {
		t.Fatalf(
			"PC efetivo esperado 12000, recebido %d",
			participant.EffectivePower,
		)
	}

	if participant.Bonuses.BossDamagePercent != 20 {
		t.Fatalf(
			"Boss Damage esperado 20, recebido %d",
			participant.Bonuses.BossDamagePercent,
		)
	}

	if participant.Bonuses.CrystalRewardPercent != 15 {
		t.Fatalf(
			"Crystal Reward esperado 15, recebido %d",
			participant.Bonuses.CrystalRewardPercent,
		)
	}

	if participant.Bonuses.StellarStoneChancePercent != 4 {
		t.Fatalf(
			"Stellar Stone esperado 4, recebido %d",
			participant.Bonuses.StellarStoneChancePercent,
		)
	}

	if participant.Bonuses.BlessingChancePercent != 3 {
		t.Fatalf(
			"Blessing esperado 3, recebido %d",
			participant.Bonuses.BlessingChancePercent,
		)
	}
}

func TestRaidParticipantRejectsInvalidPower(
	t *testing.T,
) {
	tests :=
		[]*EquipmentSummary{
			nil,
			{
				CombatPower: 0,
			},
			{
				CombatPower: -1,
			},
		}

	for _, summary := range tests {

		_, err :=
			RaidParticipantFromEquipmentSummary(
				"player-a",
				"Player A",
				summary,
			)

		if !errors.Is(
			err,
			ErrRaidInvalidParticipant,
		) {
			t.Fatalf(
				"esperado ErrRaidInvalidParticipant, recebido %v",
				err,
			)
		}
	}
}

func TestOpenRaidForGroupCreatesHistoryAndSession(
	t *testing.T,
) {
	setupRaidHistoryTestDB(t)

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	now :=
		time.Date(
			2026,
			9,
			30,
			12,
			0,
			0,
			0,
			time.UTC,
		)

	result, err :=
		OpenRaidForGroup(
			catalog,
			"group-a",
			now,
		)

	if err != nil {
		t.Fatal(err)
	}

	if result == nil ||
		result.Session == nil {

		t.Fatal(
			"resultado da Raid não deveria ser nil",
		)
	}

	if result.Boss.ID == "" {
		t.Fatal(
			"Boss sorteado está vazio",
		)
	}

	if result.Session.Boss.ID !=
		result.Boss.ID {

		t.Fatalf(
			"Boss da sessão %s difere do sorteado %s",
			result.Session.Boss.ID,
			result.Boss.ID,
		)
	}

	active, err :=
		GetActiveRaidSession(
			"group-a",
		)

	if err != nil {
		t.Fatal(err)
	}

	if active.ID !=
		result.Session.ID {

		t.Fatalf(
			"Raid ativa esperada %d, recebida %d",
			result.Session.ID,
			active.ID,
		)
	}

	recent, err :=
		RecentRaidBossIDs(
			"group-a",
			result.Boss.Rarity,
			10,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(recent) != 1 {
		t.Fatalf(
			"esperado 1 Boss no histórico, recebido %d",
			len(recent),
		)
	}

	if recent[0] !=
		result.Boss.ID {

		t.Fatalf(
			"histórico esperado %s, recebido %s",
			result.Boss.ID,
			recent[0],
		)
	}
}

func TestOpenRaidRejectsSecondActiveRaid(
	t *testing.T,
) {
	setupRaidHistoryTestDB(t)

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	now :=
		time.Now()

	if _, err :=
		OpenRaidForGroup(
			catalog,
			"group-a",
			now,
		); err != nil {

		t.Fatal(err)
	}

	_, err =
		OpenRaidForGroup(
			catalog,
			"group-a",
			now.Add(
				time.Minute,
			),
		)

	if !errors.Is(
		err,
		ErrRaidActiveExists,
	) {
		t.Fatalf(
			"esperado ErrRaidActiveExists, recebido %v",
			err,
		)
	}
}

func TestOpenRaidDifferentGroupsAreIndependent(
	t *testing.T,
) {
	setupRaidHistoryTestDB(t)

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	now :=
		time.Now()

	first, err :=
		OpenRaidForGroup(
			catalog,
			"group-a",
			now,
		)

	if err != nil {
		t.Fatal(err)
	}

	second, err :=
		OpenRaidForGroup(
			catalog,
			"group-b",
			now,
		)

	if err != nil {
		t.Fatal(err)
	}

	if first.Session.GroupJID !=
		"group-a" {

		t.Fatalf(
			"grupo inesperado: %s",
			first.Session.GroupJID,
		)
	}

	if second.Session.GroupJID !=
		"group-b" {

		t.Fatalf(
			"grupo inesperado: %s",
			second.Session.GroupJID,
		)
	}
}
