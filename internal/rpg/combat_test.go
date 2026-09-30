package rpg

import "testing"

func TestCombatSuccessChance(
	t *testing.T,
) {
	tests := []struct {
		name string

		attacker int
		defender int

		expected int
	}{
		{
			name:     "poder igual",
			attacker: 155,
			defender: 155,
			expected: 50,
		},
		{
			name:     "atacante mais forte",
			attacker: 420,
			defender: 155,
			expected: 73,
		},
		{
			name:     "defensor mais forte",
			attacker: 155,
			defender: 420,
			expected: 26,
		},
		{
			name:     "limite superior",
			attacker: 5000,
			defender: 100,
			expected: 90,
		},
		{
			name:     "limite inferior",
			attacker: 100,
			defender: 5000,
			expected: 10,
		},
		{
			name:     "ambos sem equipamento",
			attacker: 0,
			defender: 0,
			expected: 50,
		},
		{
			name:     "atacante zero",
			attacker: 0,
			defender: 155,
			expected: 10,
		},
		{
			name:     "defensor zero",
			attacker: 155,
			defender: 0,
			expected: 90,
		},
	}

	for _, tt := range tests {

		t.Run(
			tt.name,
			func(t *testing.T) {
				got :=
					CombatSuccessChance(
						tt.attacker,
						tt.defender,
					)

				if got !=
					tt.expected {

					t.Fatalf(
						"CombatSuccessChance(%d, %d) = %d; esperado %d",
						tt.attacker,
						tt.defender,
						got,
						tt.expected,
					)
				}
			},
		)
	}
}
