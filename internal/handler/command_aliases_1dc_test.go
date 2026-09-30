package handler

import "testing"

func TestPhase1DCCommandAliases(
	t *testing.T,
) {

	tests :=
		map[string]string{
			"!pve": "!pve",

			"!cacar": "!pve",

			"!caçar": "!pve",

			"!CACAR": "!pve",

			"!lutar": "!pve",

			"!combater": "!pve",

			"!coletar": "!coletar",

			"!coleta": "!coletar",

			"!recolher": "!coletar",
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

func TestPhase1DCEquipmentAliases(
	t *testing.T,
) {

	tests :=
		map[string]string{
			"!equip": "!equipar",

			"!equipar": "!equipar",

			"!vestir": "!equipar",

			"!desequipar": "!desequipar",

			"!tirar": "!desequipar",

			"!desvestir": "!desequipar",

			"!equipamentos": "!equipamentos",

			"!equipamento": "!equipamentos",

			"!gear": "!equipamentos",
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
