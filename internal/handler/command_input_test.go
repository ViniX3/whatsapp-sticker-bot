package handler

import "testing"

func TestNormalizeCommandToken(
	t *testing.T,
) {

	tests :=
		map[string]string{
			"RUÍNAS":      "ruinas",
			"Ruínas":      "ruinas",
			"FORTALEZA":   "fortaleza",
			"!Inventário": "!inventario",
			"ÉPICO":       "epico",
			"máximo":      "maximo",
			"  Cripta  ":  "cripta",
			"!MASMORRAS":  "!masmorras",
		}

	for input, want := range tests {

		got :=
			normalizeCommandToken(
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

func TestCanonicalCommand(
	t *testing.T,
) {

	tests :=
		map[string]string{
			"!Inventário": "!inventario",
			"!inv":        "!inventario",
			"!Dungeon":    "!dungeons",
			"!masmorra":   "!dungeons",
			"!APOSTA":     "!bet",
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
