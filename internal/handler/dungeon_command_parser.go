package handler

import "strings"

type dungeonCommandIntent struct {
	Action  string
	Dungeon string
}

const (
	dungeonActionList  = "list"
	dungeonActionInfo  = "info"
	dungeonActionEnter = "enter"

	dungeonRuins    = "ruinas"
	dungeonCrypt    = "cripta"
	dungeonFortress = "fortaleza"
)

// parseDungeonCommand interpreta formas naturais do comando.
//
// Exemplos aceitos:
//
//	!dungeons
//	!dungeon
//	!masmorra
//
//	!dungeons ruinas
//	!dungeons ruínas
//	!dungeons fortaleza
//
//	!dungeons ruinas entrar
//	!dungeons entrar ruinas
//	!dungeons ruínas entrar
//
//	!entrar ruinas
//	!entrar fortaleza
//	!entrar cripta
func parseDungeonCommand(
	text string,
) (
	dungeonCommandIntent,
	bool,
) {

	parts :=
		strings.Fields(
			text,
		)

	if len(parts) == 0 {
		return dungeonCommandIntent{},
			false
	}

	command :=
		canonicalCommand(
			parts[0],
		)

	// !entrar é uma forma curta específica.
	if normalizeCommandToken(
		parts[0],
	) == "!entrar" {

		return parseShortDungeonEnter(
			parts,
		)
	}

	if command != "!dungeons" {
		return dungeonCommandIntent{},
			false
	}

	if len(parts) == 1 {
		return dungeonCommandIntent{
				Action: dungeonActionList,
			},
			true
	}

	var (
		dungeon string
		action  string
	)

	for _, raw := range parts[1:] {

		token :=
			normalizeCommandToken(
				raw,
			)

		if parsedDungeon,
			ok :=
			parseDungeonName(
				token,
			); ok {

			// Duas dungeons diferentes na mesma mensagem:
			// entrada ambígua.
			if dungeon != "" &&
				dungeon != parsedDungeon {

				return dungeonCommandIntent{},
					false
			}

			dungeon =
				parsedDungeon

			continue
		}

		if parsedAction,
			ok :=
			parseDungeonAction(
				token,
			); ok {

			if action != "" &&
				action != parsedAction {

				return dungeonCommandIntent{},
					false
			}

			action =
				parsedAction

			continue
		}

		// Palavra desconhecida.
		return dungeonCommandIntent{},
			false
	}

	if dungeon == "" {
		return dungeonCommandIntent{},
			false
	}

	// Somente o nome da dungeon:
	//
	// !dungeons fortaleza
	//
	// mantém comportamento de consulta/detalhes.
	if action == "" {
		action =
			dungeonActionInfo
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
		return dungeonCommandIntent{},
			false
	}

	dungeon,
		ok :=
		parseDungeonName(
			normalizeCommandToken(
				parts[1],
			),
		)

	if !ok {
		return dungeonCommandIntent{},
			false
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

	switch normalizeCommandToken(
		value,
	) {

	case "ruina",
		"ruinas",
		"ruin",
		"ruins":

		return dungeonRuins,
			true

	case "cripta",
		"criptas",
		"crypt",
		"crypts":

		return dungeonCrypt,
			true

	case "fortaleza",
		"fortalezas",
		"fort",
		"fortress":

		return dungeonFortress,
			true
	}

	return "",
		false
}

func parseDungeonAction(
	value string,
) (
	string,
	bool,
) {

	switch normalizeCommandToken(
		value,
	) {

	case "entrar",
		"entrada",
		"acessar",
		"iniciar",
		"comecar":

		return dungeonActionEnter,
			true

	case "info",
		"detalhes",
		"detalhe",
		"ver":

		return dungeonActionInfo,
			true
	}

	return "",
		false
}
