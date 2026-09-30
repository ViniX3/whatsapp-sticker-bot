package rpg

import "testing"

func TestApplyPVECrystalRewardBonus(
	t *testing.T,
) {

	tests := []struct {
		name     string
		reward   int
		bonus    int
		expected int
	}{
		{
			name:     "sem bonus",
			reward:   10,
			bonus:    0,
			expected: 10,
		},
		{
			name:     "bonus 20 por cento",
			reward:   10,
			bonus:    20,
			expected: 12,
		},
		{
			name:     "bonus 50 por cento",
			reward:   40,
			bonus:    50,
			expected: 60,
		},
		{
			name:     "bonus minimo gera um cristal",
			reward:   1,
			bonus:    10,
			expected: 2,
		},
		{
			name:     "recompensa zero",
			reward:   0,
			bonus:    50,
			expected: 0,
		},
	}

	for _, test := range tests {

		t.Run(
			test.name,
			func(t *testing.T) {

				got :=
					applyPVECrystalRewardBonus(
						test.reward,
						test.bonus,
					)

				if got !=
					test.expected {

					t.Fatalf(
						"esperado %d, obtido %d",
						test.expected,
						got,
					)
				}
			},
		)
	}
}
