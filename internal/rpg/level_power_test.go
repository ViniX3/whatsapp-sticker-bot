package rpg

import "testing"

func TestLevelCombatPowerBonus(
	t *testing.T,
) {

	tests := []struct {
		level    int
		expected int
	}{
		// Limites inferiores.
		{level: -1, expected: 0},
		{level: 0, expected: 0},
		{level: 1, expected: 0},

		// Faixa 1.
		{level: 2, expected: 10},
		{level: 100, expected: 990},

		// Faixa 2.
		{level: 101, expected: 1010},
		{level: 250, expected: 3990},

		// Faixa 3.
		{level: 251, expected: 4015},
		{level: 400, expected: 7740},

		// Faixa 4.
		{level: 401, expected: 7770},
		{level: 600, expected: 13740},

		// Faixa 5.
		{level: 601, expected: 13775},
		{level: 750, expected: 18990},

		// Faixa 6.
		{level: 751, expected: 19030},
		{level: 850, expected: 22990},

		// Faixa 7.
		{level: 851, expected: 23035},
		{level: 950, expected: 27490},

		// Faixa 8.
		{level: 951, expected: 27540},
		{level: 1000, expected: 29990},

		// Acima do level cap continua limitado
		// ao bônus máximo atual.
		{level: 1001, expected: 29990},
		{level: 5000, expected: 29990},
	}

	for _, tt := range tests {

		got :=
			LevelCombatPowerBonus(
				tt.level,
			)

		if got != tt.expected {

			t.Fatalf(
				"LevelCombatPowerBonus(%d) = %d; esperado %d",
				tt.level,
				got,
				tt.expected,
			)
		}
	}
}
