package handler

import "testing"

func TestCanonicalAction(
	t *testing.T,
) {

	tests :=
		map[string]string{
			"guardar":   actionDeposit,
			"GUARDAR":   actionDeposit,
			"depositar": actionDeposit,
			"depósito":  actionDeposit,
			"colocar":   actionDeposit,
			"armazenar": actionDeposit,

			"retirar": actionWithdraw,
			"sacar":   actionWithdraw,
			"saque":   actionWithdraw,
			"pegar":   actionWithdraw,
			"remover": actionWithdraw,

			"tipos":  actionTypes,
			"tipo":   actionTypes,
			"cofres": actionTypes,
			"níveis": actionTypes,
		}

	for input, want := range tests {

		got,
			ok :=
			canonicalAction(
				input,
			)

		if !ok {
			t.Fatalf(
				"%q deveria ser reconhecido",
				input,
			)
		}

		if got != want {
			t.Fatalf(
				"%q: esperado %q, recebido %q",
				input,
				want,
				got,
			)
		}
	}
}

func TestCanonicalShopAndForgeActions(
	t *testing.T,
) {

	buyTests := []string{
		"comprar",
		"COMPRA",
		"adquirir",
		"Adquire",
	}

	for _, input := range buyTests {
		got,
			ok :=
			canonicalAction(
				input,
			)

		if !ok ||
			got != actionBuy {

			t.Fatalf(
				"%q: esperado buy, recebido %q / %v",
				input,
				got,
				ok,
			)
		}
	}

	craftTests := []string{
		"forjar",
		"fabricar",
		"criar",
		"produzir",
		"craft",
	}

	for _, input := range craftTests {
		got,
			ok :=
			canonicalAction(
				input,
			)

		if !ok ||
			got != actionCraft {

			t.Fatalf(
				"%q: esperado craft, recebido %q / %v",
				input,
				got,
				ok,
			)
		}
	}
}

func TestShopAndForgeCommandAliases(
	t *testing.T,
) {

	tests :=
		map[string]string{
			"!comprar":  "!comprar",
			"!adquirir": "!comprar",

			"!forjar":   "!forjar",
			"!fabricar": "!forjar",
			"!criar":    "!forjar",
			"!produzir": "!forjar",
			"!craft":    "!forjar",
		}

	for input, want := range tests {

		got :=
			canonicalCommand(
				input,
			)

		if got != want {
			t.Fatalf(
				"%q: esperado %q, recebido %q",
				input,
				want,
				got,
			)
		}
	}
}
