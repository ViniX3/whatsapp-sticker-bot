package handler

import (
	"testing"

	"whatsapp-sticker-bot/internal/rpg"
)

func TestParseDungeonCommandList(
	t *testing.T,
) {

	tests := []string{
		"!dungeons",
		"!dungeon",
		"!masmorra",
		"!masmorras",
	}

	for _, input := range tests {

		result,
			ok :=
			parseDungeonCommand(
				input,
			)

		if !ok {
			t.Fatalf(
				"%q deveria ser reconhecido",
				input,
			)
		}

		if result.Action != dungeonActionList {
			t.Fatalf(
				"%q: esperado list, recebido %q",
				input,
				result.Action,
			)
		}
	}
}

func TestParseDungeonCommandInfo(
	t *testing.T,
) {

	tests :=
		map[string]string{
			"!dungeons ruinas":    dungeonRuins,
			"!dungeons ruínas":    dungeonRuins,
			"!dungeons cripta":    dungeonCrypt,
			"!dungeons fortaleza": dungeonFortress,
			"!dungeon FORTALEZA":  dungeonFortress,
			"!masmorra ruína":     dungeonRuins,
		}

	for input, expectedDungeon := range tests {

		result,
			ok :=
			parseDungeonCommand(
				input,
			)

		if !ok {
			t.Fatalf(
				"%q deveria ser reconhecido",
				input,
			)
		}

		if result.Action != dungeonActionInfo {
			t.Fatalf(
				"%q: ação esperada info, recebido %q",
				input,
				result.Action,
			)
		}

		if result.Dungeon != expectedDungeon {
			t.Fatalf(
				"%q: dungeon esperada %q, recebida %q",
				input,
				expectedDungeon,
				result.Dungeon,
			)
		}
	}
}

func TestParseDungeonCommandEnterFlexibleOrder(
	t *testing.T,
) {

	tests :=
		map[string]string{
			"!dungeons ruinas entrar": dungeonRuins,
			"!dungeons entrar ruinas": dungeonRuins,
			"!dungeons ruínas entrar": dungeonRuins,
			"!dungeons entrar ruínas": dungeonRuins,

			"!dungeons cripta entrar": dungeonCrypt,
			"!dungeons entrar cripta": dungeonCrypt,

			"!dungeons fortaleza entrar": dungeonFortress,
			"!dungeons entrar fortaleza": dungeonFortress,

			"!dungeon iniciar fortaleza":  dungeonFortress,
			"!masmorra fortaleza acessar": dungeonFortress,

			"!entrar ruinas":    dungeonRuins,
			"!entrar ruínas":    dungeonRuins,
			"!entrar cripta":    dungeonCrypt,
			"!entrar fortaleza": dungeonFortress,
		}

	for input, expectedDungeon := range tests {

		result,
			ok :=
			parseDungeonCommand(
				input,
			)

		if !ok {
			t.Fatalf(
				"%q deveria ser reconhecido",
				input,
			)
		}

		if result.Action != dungeonActionEnter {
			t.Fatalf(
				"%q: ação esperada enter, recebido %q",
				input,
				result.Action,
			)
		}

		if result.Dungeon != expectedDungeon {
			t.Fatalf(
				"%q: dungeon esperada %q, recebida %q",
				input,
				expectedDungeon,
				result.Dungeon,
			)
		}
	}
}

func TestParseDungeonCommandRejectsInvalidInput(
	t *testing.T,
) {

	tests := []string{
		"!entrar",
		"!entrar inexistente",
		"!dungeons entrar",
		"!dungeons ruinas qualquercoisa",
		"!dungeons ruinas cripta entrar",
		"!dungeons fortaleza entrar info",
	}

	for _, input := range tests {

		_,
			ok :=
			parseDungeonCommand(
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

func TestDungeonParserProducesValidDungeonIDs(
	t *testing.T,
) {

	tests := []string{
		"!dungeons ruínas",
		"!dungeons cripta",
		"!dungeons fortaleza",
		"!entrar ruinas",
		"!dungeons entrar fortaleza",
	}

	for _, input := range tests {

		intent,
			ok :=
			parseDungeonCommand(
				input,
			)

		if !ok {
			t.Fatalf(
				"%q não foi reconhecido",
				input,
			)
		}

		if intent.Dungeon == "" {
			t.Fatalf(
				"%q não retornou dungeon",
				input,
			)
		}

		_,
			exists :=
			rpg.DungeonByID(
				intent.Dungeon,
			)

		if !exists {
			t.Fatalf(
				"%q produziu ID inválido: %q",
				input,
				intent.Dungeon,
			)
		}
	}
}
