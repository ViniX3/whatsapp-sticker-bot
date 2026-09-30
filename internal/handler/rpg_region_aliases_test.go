package handler

import (
	"testing"

	"whatsapp-sticker-bot/internal/rpg"
)

func TestParseNaturalRPGRegion(
	t *testing.T,
) {

	tests :=
		map[string]rpg.GatheringRegion{
			"floresta": rpg.GatheringForest,

			"FLORESTA": rpg.GatheringForest,

			"forest": rpg.GatheringForest,

			"mata": rpg.GatheringForest,

			"bosque": rpg.GatheringForest,

			"pedreira": rpg.GatheringQuarry,

			"quarry": rpg.GatheringQuarry,

			"rocha": rpg.GatheringQuarry,

			"rochas": rpg.GatheringQuarry,

			"mina": rpg.GatheringMine,

			"mine": rpg.GatheringMine,

			"minerar": rpg.GatheringMine,

			"mineração": rpg.GatheringMine,
		}

	for input, want := range tests {

		got,
			ok :=
			parseNaturalRPGRegion(
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

func TestParseNaturalRPGRegionRejectsUnknown(
	t *testing.T,
) {

	for _, input := range []string{
		"cidade",
		"oceano",
		"deserto",
		"qualquercoisa",
	} {

		_,
			ok :=
			parseNaturalRPGRegion(
				input,
			)

		if ok {
			t.Fatalf(
				"%q não deveria ser aceito",
				input,
			)
		}
	}
}
