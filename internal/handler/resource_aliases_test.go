package handler

import (
	"testing"

	"whatsapp-sticker-bot/internal/rpg"
)

func TestCanonicalVaultResource(
	t *testing.T,
) {

	tests :=
		map[string]rpg.VaultResource{
			"gold": rpg.VaultResourceGold,

			"ouro": rpg.VaultResourceGold,

			"cristal": rpg.VaultResourceMagicCrystals,

			"cristais": rpg.VaultResourceMagicCrystals,

			"cristais mágicos": rpg.VaultResourceMagicCrystals,

			"CRISTAIS MÁGICOS": rpg.VaultResourceMagicCrystals,

			"pedra": rpg.VaultResourceStellarStones,

			"pedras": rpg.VaultResourceStellarStones,

			"pedra estelar": rpg.VaultResourceStellarStones,

			"pedras estelares": rpg.VaultResourceStellarStones,
		}

	for input, want := range tests {

		got,
			ok :=
			canonicalVaultResource(
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

func TestCanonicalVaultResourceRejectsInvalid(
	t *testing.T,
) {

	for _, input := range []string{
		"diamante",
		"madeira",
		"mana",
	} {

		_,
			ok :=
			canonicalVaultResource(
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
