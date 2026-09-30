package handler

import (
	"strings"

	"whatsapp-sticker-bot/internal/rpg"
)

// canonicalVaultResource interpreta nomes naturais de recursos.
//
// Pode receber uma ou mais palavras:
//
//	gold
//	ouro
//	cristais
//	cristais mágicos
//	pedras
//	pedras estelares
func canonicalVaultResource(
	value string,
) (
	rpg.VaultResource,
	bool,
) {

	parts :=
		strings.Fields(
			value,
		)

	normalizedParts :=
		make(
			[]string,
			0,
			len(parts),
		)

	for _, part := range parts {

		normalizedParts =
			append(
				normalizedParts,
				normalizeCommandToken(
					part,
				),
			)
	}

	normalized :=
		strings.Join(
			normalizedParts,
			" ",
		)

	switch normalized {

	// ========================================================
	// GOLD
	// ========================================================

	case "gold",
		"ouro":

		return rpg.VaultResourceGold,
			true

	// ========================================================
	// CRISTAIS MÁGICOS
	// ========================================================

	case "cristal",
		"cristais",
		"cristal magico",
		"cristais magicos",
		"crystal",
		"crystals":

		return rpg.VaultResourceMagicCrystals,
			true

	// ========================================================
	// PEDRAS ESTELARES
	// ========================================================

	case "pedra",
		"pedras",
		"pedra estelar",
		"pedras estelares",
		"estelar",
		"estelares",
		"stellar stone",
		"stellar stones":

		return rpg.VaultResourceStellarStones,
			true
	}

	return "",
		false
}
