package handler

import (
	"errors"
	"testing"
)

func TestParseFlexibleGoldAmount(
	t *testing.T,
) {

	const balance = 1_000_000

	tests :=
		[]struct {
			input string
			want  int
		}{
			{
				input: "1000",
				want:  1000,
			},
			{
				input: "1.000",
				want:  1000,
			},
			{
				input: "1_000",
				want:  1000,
			},
			{
				input: "all",
				want:  1_000_000,
			},
			{
				input: "ALL",
				want:  1_000_000,
			},
			{
				input: "tudo",
				want:  1_000_000,
			},
			{
				input: "máximo",
				want:  1_000_000,
			},
			{
				input: "saldo",
				want:  1_000_000,
			},
			{
				input: "metade",
				want:  500_000,
			},
			{
				input: "half",
				want:  500_000,
			},
			{
				input: "20%",
				want:  200_000,
			},
			{
				input: "50%",
				want:  500_000,
			},
			{
				input: "100%",
				want:  1_000_000,
			},
		}

	for _, test := range tests {

		t.Run(
			test.input,
			func(t *testing.T) {

				got,
					err :=
					parseFlexibleGoldAmount(
						test.input,
						balance,
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

func TestParseFlexibleGoldAmountPercentValidation(
	t *testing.T,
) {

	for _, input := range []string{
		"0%",
		"101%",
		"500%",
	} {

		_,
			err :=
			parseFlexibleGoldAmount(
				input,
				100_000,
			)

		if !errors.Is(
			err,
			errFlexibleAmountPercentRange,
		) {

			t.Fatalf(
				"%s: erro esperado de percentual, recebido %v",
				input,
				err,
			)
		}
	}
}

func TestParseFlexibleGoldAmountRequiresBalance(
	t *testing.T,
) {

	for _, input := range []string{
		"all",
		"tudo",
		"metade",
		"20%",
	} {

		_,
			err :=
			parseFlexibleGoldAmount(
				input,
				0,
			)

		if !errors.Is(
			err,
			errFlexibleAmountNoBalance,
		) {

			t.Fatalf(
				"%s: deveria exigir saldo",
				input,
			)
		}
	}
}
