package gold

import "testing"

func TestSafeGoldPercentLargeAmounts(
	t *testing.T,
) {

	tests := []struct {
		name    string
		amount  int
		percent int
		want    int
	}{
		{
			name:    "1 quatrilhao 100x",
			amount:  1_000_000_000_000_000,
			percent: 10_000,
			want:    100_000_000_000_000_000,
		},
		{
			name:    "2 quatrilhoes 100x",
			amount:  2_000_000_000_000_000,
			percent: 10_000,
			want:    200_000_000_000_000_000,
		},
		{
			name:    "5 quatrilhoes 20x",
			amount:  5_000_000_000_000_000,
			percent: 2_000,
			want:    100_000_000_000_000_000,
		},
		{
			name:    "6 quatrilhoes 20x",
			amount:  6_000_000_000_000_000,
			percent: 2_000,
			want:    120_000_000_000_000_000,
		},
		{
			name:    "6 quatrilhoes 100x",
			amount:  6_000_000_000_000_000,
			percent: 10_000,
			want:    600_000_000_000_000_000,
		},
	}

	for _, test := range tests {

		t.Run(
			test.name,
			func(t *testing.T) {

				got,
					err :=
					safeGoldPercent(
						test.amount,
						test.percent,
					)

				if err != nil {
					t.Fatalf(
						"erro inesperado: %v",
						err,
					)
				}

				if got != test.want {
					t.Fatalf(
						"esperado %d, recebido %d",
						test.want,
						got,
					)
				}
			},
		)
	}
}

func TestSafeGoldPercentDetectsOverflow(
	t *testing.T,
) {

	maxInt :=
		int(^uint(0) >> 1)

	_,
		err :=
		safeGoldPercent(
			maxInt,
			200,
		)

	if err == nil {
		t.Fatal(
			"era esperado erro de overflow",
		)
	}
}

func TestSafeGoldAddDetectsOverflow(
	t *testing.T,
) {

	maxInt :=
		int(^uint(0) >> 1)

	_,
		err :=
		safeGoldAdd(
			maxInt,
			1,
		)

	if err == nil {
		t.Fatal(
			"era esperado erro de overflow",
		)
	}
}
