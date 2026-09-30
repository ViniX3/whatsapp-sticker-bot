package rpg

import "testing"

func TestLevelCombatPowerBonus(
	t *testing.T,
) {

	tests := []struct {
		level int
		want  int
	}{
		{1, 0},
		{2, 10},
		{10, 90},
		{50, 490},
		{100, 990},
		{150, 1490},
		{200, 1990},
	}

	for _, test := range tests {

		got :=
			LevelCombatPowerBonus(
				test.level,
			)

		if got != test.want {

			t.Fatalf(
				"nível %d: esperado +%d PC, recebido +%d PC",
				test.level,
				test.want,
				got,
			)
		}
	}
}
