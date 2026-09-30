package handler

import "testing"

func TestNormalizeCommandText(
	t *testing.T,
) {

	tests :=
		map[string]string{
			"!INVENTÁRIO": "!inventario",

			"!Inventário 2": "!inventario 2",

			"!APOSTA 20%": "!bet 20%",

			"!Apostar tudo": "!bet tudo",

			"!DUNGEON entrar Ruínas": "!dungeons entrar Ruínas",

			"!MASMORRA fortaleza": "!dungeons fortaleza",

			// Contextual: não deve virar
			// "!dungeons Ruínas".
			"!Entrar Ruínas": "!entrar Ruínas",

			"mensagem normal": "mensagem normal",
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

func TestSuggestCommand(
	t *testing.T,
) {

	tests :=
		map[string]string{
			"!dungeos":   "!dungeons",
			"!inventaro": "!inventario",
			"!slost":     "!slots",
			"!raking":    "!ranking",
			"!conquitsa": "!conquistas",
			"!cristas":   "!cristais",
		}

	for input, want := range tests {

		got,
			ok :=
			suggestCommand(
				input,
			)

		if !ok {
			t.Fatalf(
				"%q deveria gerar sugestão",
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

func TestSuggestCommandRejectsDistantInput(
	t *testing.T,
) {

	tests := []string{
		"!abcdef",
		"!qualquercoisa",
		"!xyzxyzxyz",
	}

	for _, input := range tests {

		got,
			ok :=
			suggestCommand(
				input,
			)

		if ok {
			t.Fatalf(
				"%q não deveria sugerir %q",
				input,
				got,
			)
		}
	}
}

func TestSuggestCommandNeverExecutesAlias(
	t *testing.T,
) {

	// A sugestão pode encontrar !bet,
	// mas o retorno é apenas texto.
	got,
		ok :=
		suggestCommand(
			"!bett",
		)

	if !ok ||
		got != "!bet" {

		t.Fatalf(
			"esperado sugestão !bet, recebido %q / %v",
			got,
			ok,
		)
	}
}
