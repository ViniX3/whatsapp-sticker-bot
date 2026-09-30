package handler

import (
	"strings"
	"testing"
)

func TestPhase1DEMenuDiscoverability(
	t *testing.T,
) {

	required := []string{
		"v3.5.0",

		"!perfil",
		"!ficha",
		"!personagem",

		"!cofre",

		"!inventario",
		"!mochila",
		"!bolsa",

		"!equipamentos",
		"!arsenal",

		"!coletar",
		"!recolher",

		"!pve",
		"!caçar",

		"!dungeons",
		"!entrar",

		"!cristais",

		"!loja",
		"!comprar",
		"!adquirir",

		"!forja",
		"!forjar",
		"!fabricar",

		"!equipar",
		"!vestir",

		"!desequipar",
		"!tirar",
	}

	for _, expected := range required {

		if !strings.Contains(
			menuMessage,
			expected,
		) {

			t.Fatalf(
				"menu não contém %q",
				expected,
			)
		}
	}
}
