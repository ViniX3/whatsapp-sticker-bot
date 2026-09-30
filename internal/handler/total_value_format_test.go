package handler

import "testing"

func TestFormatTotalValue(
	t *testing.T,
) {

	tests := []struct {
		value    int
		resource string
		want     string
	}{
		{
			999,
			"Gold",
			"999 Gold",
		},
		{
			1_000,
			"Gold",
			"1.000 mil de Gold",
		},
		{
			80_598,
			"Gold",
			"80.598 mil de Gold",
		},
		{
			132_587,
			"Gold",
			"132.587 mil de Gold",
		},
		{
			1_325_157,
			"Gold",
			"1.325.157 milhão de Gold",
		},
		{
			80_598,
			"Cristais",
			"80.598 mil de Cristais",
		},
		{
			1_325_157,
			"Cristais",
			"1.325.157 milhão de Cristais",
		},
		{
			2_112_534_534_284_474_626,
			"Gold",
			"2.112.534.534.284.474.626 quintilhão de Gold",
		},
	}

	for _, test := range tests {

		got :=
			formatTotalValue(
				test.value,
				test.resource,
			)

		if got != test.want {
			t.Fatalf(
				"%d: esperado %q, recebido %q",
				test.value,
				test.want,
				got,
			)
		}
	}
}
