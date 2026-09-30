package rpg

import (
	"testing"
	"time"
)

func TestRaidConstants(
	t *testing.T,
) {

	if RaidMaxParticipants != 10 {
		t.Fatalf(
			"esperado máximo de 10 participantes, recebido %d",
			RaidMaxParticipants,
		)
	}

	if RaidJoinWindow != 90*time.Second {
		t.Fatalf(
			"esperada janela de 1m30s, recebida %s",
			RaidJoinWindow,
		)
	}
}

func TestRaidBonusesFromSets(
	t *testing.T,
) {

	active :=
		[]ActiveSetBonus{
			{
				Effects: []SetEffect{
					{
						Type: BonusBossDamagePercent,

						Value: 12,
					},
					{
						Type: BonusCrystalRewardPercent,

						Value: 15,
					},
				},
			},
			{
				Effects: []SetEffect{
					{
						Type: BonusBossDamagePercent,

						Value: 8,
					},
					{
						Type: BonusBlessingChancePercent,

						Value: 4,
					},
					{
						Type: BonusStellarStoneChancePercent,

						Value: 3,
					},
				},
			},
		}

	bonuses :=
		RaidBonusesFromSets(
			active,
		)

	if bonuses.BossDamagePercent != 20 {
		t.Fatalf(
			"esperado +20%% contra Boss, recebido %d%%",
			bonuses.BossDamagePercent,
		)
	}

	if bonuses.CrystalRewardPercent != 15 {
		t.Fatalf(
			"esperado +15%% Cristais, recebido %d%%",
			bonuses.CrystalRewardPercent,
		)
	}

	if bonuses.BlessingChancePercent != 4 {
		t.Fatalf(
			"esperado +4%% Bênção, recebido %d%%",
			bonuses.BlessingChancePercent,
		)
	}

	if bonuses.StellarStoneChancePercent != 3 {
		t.Fatalf(
			"esperado +3%% Pedra Estelar, recebido %d%%",
			bonuses.StellarStoneChancePercent,
		)
	}
}

func TestRaidEffectiveBossPower(
	t *testing.T,
) {

	tests :=
		[]struct {
			power int
			bonus int
			want  int
		}{
			{
				power: 1000,
				bonus: 0,
				want:  1000,
			},
			{
				power: 1000,
				bonus: 10,
				want:  1100,
			},
			{
				power: 2500,
				bonus: 20,
				want:  3000,
			},
		}

	for _, test := range tests {

		got :=
			RaidEffectiveBossPower(
				test.power,
				test.bonus,
			)

		if got != test.want {
			t.Fatalf(
				"PC %d + %d%%: esperado %d, recebido %d",
				test.power,
				test.bonus,
				test.want,
				got,
			)
		}
	}
}

func TestRaidTotalPower(
	t *testing.T,
) {

	participants :=
		[]RaidParticipantPower{
			{
				CombatPower: 1000,

				Bonuses: RaidBonuses{
					BossDamagePercent: 10,
				},
			},
			{
				CombatPower: 2000,
			},
			{
				CombatPower: 1500,

				Bonuses: RaidBonuses{
					BossDamagePercent: 20,
				},
			},
		}

	// 1100 + 2000 + 1800
	want := 4900

	got :=
		RaidTotalPower(
			participants,
		)

	if got != want {
		t.Fatalf(
			"esperado poder total %d, recebido %d",
			want,
			got,
		)
	}
}

func TestRaidSuccessChance(
	t *testing.T,
) {

	tests :=
		[]struct {
			raid int
			boss int
			want int
		}{
			{
				raid: 10000,
				boss: 10000,
				want: 50,
			},
			{
				raid: 20000,
				boss: 10000,
				want: 67,
			},
			{
				raid: 30000,
				boss: 10000,
				want: 75,
			},
			{
				raid: 10000,
				boss: 30000,
				want: 25,
			},
		}

	for _, test := range tests {

		got :=
			RaidSuccessChance(
				test.raid,
				test.boss,
			)

		if got != test.want {
			t.Fatalf(
				"RAID %d / Boss %d: esperado %d%%, recebido %d%%",
				test.raid,
				test.boss,
				test.want,
				got,
			)
		}
	}
}

func TestRaidSuccessChanceRejectsInvalidPower(
	t *testing.T,
) {

	if got :=
		RaidSuccessChance(
			0,
			1000,
		); got != 0 {

		t.Fatalf(
			"esperado 0, recebido %d",
			got,
		)
	}

	if got :=
		RaidSuccessChance(
			1000,
			0,
		); got != 0 {

		t.Fatalf(
			"esperado 0, recebido %d",
			got,
		)
	}
}
