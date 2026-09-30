package handler

import "testing"

func TestFormatGoldAmount(
	t *testing.T,
) {
	tests :=
		[]struct {
			value int
			want  string
		}{
			{999, "999"},
			{1000, "1 mil"},
			{12500, "12,5 mil"},
			{999999, "1000 mil"},
			{1000000, "1 milhão"},
			{1250000, "1,25 milhões"},
			{1000000000, "1 bilhão"},
			{2500000000, "2,5 bilhões"},
			{1000000000000, "1 trilhão"},
			{1000000000000000, "1 quatrilhão"},
			{
				30000000000000000,
				"30 quatrilhões",
			},
		}

	for _, test := range tests {

		got :=
			formatGoldAmount(
				test.value,
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
