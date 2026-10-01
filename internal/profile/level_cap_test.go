package profile

import "testing"

func TestMaxPlayerLevel(
	t *testing.T,
) {
	info :=
		LevelFromXP(
			1000000000,
		)

	if info.Level != MaxPlayerLevel {
		t.Fatalf(
			"esperado nível máximo %d, recebido %d",
			MaxPlayerLevel,
			info.Level,
		)
	}

	if info.ProgressPercent != 100 {
		t.Fatalf(
			"nível máximo deveria estar em 100%%, recebido %d%%",
			info.ProgressPercent,
		)
	}

	if info.RequiredLevelXP != 0 {
		t.Fatalf(
			"nível máximo não deveria exigir XP adicional, recebido %d",
			info.RequiredLevelXP,
		)
	}
}

func TestLegacyLevelThresholdPreserved(
	t *testing.T,
) {
	levels := []int{
		1,
		2,
		10,
		50,
		100,
		150,
		199,
		200,
	}

	for _, level := range levels {
		expected := 0

		if level > 1 {
			expected =
				25 *
					(level - 1) *
					(level + 2)
		}

		got :=
			levelThreshold(
				level,
			)

		if got != expected {
			t.Fatalf(
				"nível %d: threshold antigo deveria ser %d, recebido %d",
				level,
				expected,
				got,
			)
		}
	}
}

func TestLevel200IsNoLongerCap(
	t *testing.T,
) {
	info :=
		LevelFromXP(
			levelThreshold(200),
		)

	if info.Level != 200 {
		t.Fatalf(
			"esperado nível 200, recebido %d",
			info.Level,
		)
	}

	if info.RequiredLevelXP != 10050 {
		t.Fatalf(
			"200 -> 201 deveria exigir 10050 XP, recebido %d",
			info.RequiredLevelXP,
		)
	}

	if info.ProgressPercent != 0 {
		t.Fatalf(
			"nível 200 recém atingido deveria iniciar em 0%%, recebido %d%%",
			info.ProgressPercent,
		)
	}
}

func TestProgressiveLevelThresholds(
	t *testing.T,
) {
	previous :=
		levelThreshold(
			200,
		)

	for level := 201; level <= MaxPlayerLevel; level++ {
		current :=
			levelThreshold(
				level,
			)

		if current <= previous {
			t.Fatalf(
				"threshold não cresceu no nível %d: anterior=%d atual=%d",
				level,
				previous,
				current,
			)
		}

		previous = current
	}
}

func TestExactLevelBoundaries(
	t *testing.T,
) {
	levels := []int{
		200,
		201,
		250,
		400,
		600,
		750,
		850,
		950,
		999,
		1000,
	}

	for _, level := range levels {
		info :=
			LevelFromXP(
				levelThreshold(level),
			)

		if info.Level != level {
			t.Fatalf(
				"threshold do nível %d retornou nível %d",
				level,
				info.Level,
			)
		}
	}
}
