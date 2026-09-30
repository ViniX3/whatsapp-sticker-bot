package rpg

import (
	"testing"
	"time"
)

func TestGatheringQoLConstants(
	t *testing.T,
) {
	if GatheringCooldown !=
		30*time.Second {

		t.Fatalf(
			"cooldown esperado de 30s, recebido %s",
			GatheringCooldown,
		)
	}

	if GatheringSweepRollsPerRegion != 2 {
		t.Fatalf(
			"esperadas 2 rolagens base por região, recebido %d",
			GatheringSweepRollsPerRegion,
		)
	}

	if GatheringSweepBonusRollChancePercent != 30 {
		t.Fatalf(
			"chance bônus esperada de 30%%, recebida %d%%",
			GatheringSweepBonusRollChancePercent,
		)
	}
}

func TestGatheringSweepRegions(
	t *testing.T,
) {
	expected :=
		[]GatheringRegion{
			GatheringForest,
			GatheringQuarry,
			GatheringMine,
		}

	if len(gatheringSweepRegions) !=
		len(expected) {

		t.Fatalf(
			"esperadas %d regiões, recebidas %d",
			len(expected),
			len(gatheringSweepRegions),
		)
	}

	for index, region := range expected {

		if gatheringSweepRegions[index] !=
			region {

			t.Fatalf(
				"região %d esperada %s, recebida %s",
				index,
				region,
				gatheringSweepRegions[index],
			)
		}
	}
}
