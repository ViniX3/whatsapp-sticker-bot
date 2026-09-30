package handler

import "whatsapp-sticker-bot/internal/rpg"

// parseNaturalRPGRegion centraliza nomes naturais para as
// regiões utilizadas por Coleta e PvE.
//
// Mantemos apenas aliases inequívocos para evitar que o bot
// interprete comandos de forma inesperada.
func parseNaturalRPGRegion(
	value string,
) (
	rpg.GatheringRegion,
	bool,
) {

	switch normalizeCommandToken(
		value,
	) {

	// ========================================================
	// FLORESTA
	// ========================================================

	case "floresta",
		"forest",
		"mata",
		"bosque":

		return rpg.GatheringForest,
			true

	// ========================================================
	// PEDREIRA
	// ========================================================

	case "pedreira",
		"quarry",
		"rocha",
		"rochas":

		return rpg.GatheringQuarry,
			true

	// ========================================================
	// MINA
	// ========================================================

	case "mina",
		"mine",
		"minerar",
		"mineracao":

		return rpg.GatheringMine,
			true
	}

	return "",
		false
}
