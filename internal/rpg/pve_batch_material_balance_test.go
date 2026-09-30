package rpg

import "testing"

func TestPVEBatchMaterialRollInterval(
	t *testing.T,
) {
	tests :=
		[]struct {
			battles int
			want    int
		}{
			{1, 1},
			{10, 1},
			{11, 2},
			{50, 2},
			{51, 4},
			{100, 4},
			{101, 6},
			{250, 6},
			{251, 10},
			{500, 10},
			{501, 20},
			{1000, 20},
		}

	for _, test := range tests {
		got :=
			pveBatchMaterialRollInterval(
				test.battles,
			)

		if got != test.want {
			t.Fatalf(
				"battles=%d: esperado intervalo %d, recebido %d",
				test.battles,
				test.want,
				got,
			)
		}
	}
}
