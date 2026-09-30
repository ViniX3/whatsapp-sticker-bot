package handler

import "testing"

func TestPhase1DDCommandAliases(
	t *testing.T,
) {

	tests :=
		map[string]string{
			"!inventario": "!inventario",

			"!inv": "!inventario",

			"!mochila": "!inventario",

			"!bolsa": "!inventario",

			"!equipamentos": "!equipamentos",

			"!arsenal": "!equipamentos",

			"!perfil": "!perfil",

			"!ficha": "!perfil",

			"!personagem": "!perfil",
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

func TestPhase1DDAliasesPreserveArguments(
	t *testing.T,
) {

	tests :=
		map[string]string{
			"!mochila 2": "!inventario 2",

			"!mochila raro": "!inventario raro",

			"!bolsa épico 2": "!inventario épico 2",

			"!ficha @5511999999999": "!perfil @5511999999999",

			"!personagem @5511999999999": "!perfil @5511999999999",
		}

	for input, want := range tests {

		got :=
			normalizeCommandText(
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
