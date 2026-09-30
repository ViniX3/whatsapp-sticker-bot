package rpg

import "testing"

func TestRaidScalePowerKeepsBaseFloor(
	t *testing.T,
) {
	got :=
		RaidScalePowerFromReference(
			12000,
			RarityLegendary,
			1000,
		)

	if got != 12000 {
		t.Fatalf(
			"esperado piso 12000, recebido %d",
			got,
		)
	}
}

func TestRaidScalePowerLegendary(
	t *testing.T,
) {
	got :=
		RaidScalePowerFromReference(
			12000,
			RarityLegendary,
			5000,
		)

	if got != 25000 {
		t.Fatalf(
			"esperado 25000, recebido %d",
			got,
		)
	}
}

func TestRaidScalePowerMythic(
	t *testing.T,
) {
	got :=
		RaidScalePowerFromReference(
			40000,
			RarityMythic,
			10000,
		)

	if got != 60000 {
		t.Fatalf(
			"esperado 60000, recebido %d",
			got,
		)
	}
}

func TestRaidScalePowerSacred(
	t *testing.T,
) {
	got :=
		RaidScalePowerFromReference(
			180000,
			RaritySacred,
			30000,
		)

	if got != 240000 {
		t.Fatalf(
			"esperado 240000, recebido %d",
			got,
		)
	}
}
