package handler

import "strings"

type dungeonCommandIntent struct {
	Action  string
	Dungeon string
}

const (
	dungeonActionList      = "list"
	dungeonActionChaosList = "chaos-list"
	dungeonActionInfo      = "info"
	dungeonActionEnter     = "enter"

	dungeonRuins         = "ruinas"
	dungeonCrypt         = "cripta"
	dungeonFortress      = "fortaleza"
	dungeonTemple        = "templo"
	dungeonLabyrinth     = "labirinto"
	dungeonThrone        = "trono"
	dungeonChaosRift     = "fenda-caos"
	dungeonVoidCathedral = "catedral-vazio"
	dungeonEndHeart      = "coracao-fim"
)

func parseDungeonCommand(
	text string,
) (
	dungeonCommandIntent,
	bool,
) {
	parts := strings.Fields(text)

	if len(parts) == 0 {
		return dungeonCommandIntent{}, false
	}

	command :=
		canonicalCommand(
			parts[0],
		)

	if normalizeCommandToken(parts[0]) == "!entrar" {
		return parseShortDungeonEnter(parts)
	}

	if command != "!dungeons" {
		return dungeonCommandIntent{}, false
	}

	if len(parts) == 1 {
		return dungeonCommandIntent{
				Action: dungeonActionList,
			},
			true
	}

	var (
		dungeon        string
		action         string
		chaosRequested bool
	)

	for _, raw := range parts[1:] {
		token :=
			normalizeCommandToken(
				raw,
			)

		if token == "caos" ||
			token == "chaos" {
			chaosRequested = true
			continue
		}

		if parsedDungeon,
			ok :=
			parseDungeonName(
				token,
			); ok {

			if dungeon != "" &&
				dungeon != parsedDungeon {
				return dungeonCommandIntent{}, false
			}

			dungeon = parsedDungeon
			continue
		}

		if parsedAction,
			ok :=
			parseDungeonAction(
				token,
			); ok {

			if action != "" &&
				action != parsedAction {
				return dungeonCommandIntent{}, false
			}

			action = parsedAction
			continue
		}

		return dungeonCommandIntent{}, false
	}

	if dungeon == "" {
		if chaosRequested &&
			action == "" {
			return dungeonCommandIntent{
					Action: dungeonActionChaosList,
				},
				true
		}

		return dungeonCommandIntent{}, false
	}

	if action == "" {
		action = dungeonActionInfo
	}

	return dungeonCommandIntent{
			Action:  action,
			Dungeon: dungeon,
		},
		true
}

func parseShortDungeonEnter(
	parts []string,
) (
	dungeonCommandIntent,
	bool,
) {
	if len(parts) != 2 {
		return dungeonCommandIntent{}, false
	}

	dungeon,
		ok :=
		parseDungeonName(
			normalizeCommandToken(
				parts[1],
			),
		)

	if !ok {
		return dungeonCommandIntent{}, false
	}

	return dungeonCommandIntent{
			Action:  dungeonActionEnter,
			Dungeon: dungeon,
		},
		true
}

func parseDungeonName(
	value string,
) (
	string,
	bool,
) {
	switch normalizeCommandToken(value) {
	case "ruina", "ruinas", "ruin", "ruins":
		return dungeonRuins, true
	case "cripta", "criptas", "crypt", "crypts":
		return dungeonCrypt, true
	case "fortaleza", "fortalezas", "fort", "fortress":
		return dungeonFortress, true
	case "templo", "abismo", "templo-abismo":
		return dungeonTemple, true
	case "labirinto", "labirintos", "rei-caido":
		return dungeonLabyrinth, true
	case "trono", "antigos", "trono-antigos":
		return dungeonThrone, true
	case "fenda", "fenda-caos":
		return dungeonChaosRift, true
	case "catedral", "vazio", "catedral-vazio":
		return dungeonVoidCathedral, true
	case "coracao", "fim", "coracao-fim":
		return dungeonEndHeart, true
	}

	return "", false
}

func parseDungeonAction(
	value string,
) (
	string,
	bool,
) {
	switch normalizeCommandToken(value) {
	case "entrar", "entrada", "acessar", "iniciar", "comecar":
		return dungeonActionEnter, true
	case "info", "detalhes", "detalhe", "ver":
		return dungeonActionInfo, true
	}

	return "", false
}
