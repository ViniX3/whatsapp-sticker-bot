package handler

import "testing"

func TestDungeonV4ParserNewNormalDungeons(
	t *testing.T,
) {
	tests :=
		map[string]string{
			"!dungeons templo":           dungeonTemple,
			"!dungeons labirinto":        dungeonLabyrinth,
			"!dungeons trono":            dungeonThrone,
			"!dungeons templo entrar":    dungeonTemple,
			"!dungeons entrar labirinto": dungeonLabyrinth,
			"!entrar trono":              dungeonThrone,
		}

	for input, expected := range tests {

		intent, ok :=
			parseDungeonCommand(
				input,
			)

		if !ok {
			t.Fatalf(
				"%q deveria ser reconhecido",
				input,
			)
		}

		if intent.Dungeon != expected {
			t.Fatalf(
				"%q: esperado %q, recebido %q",
				input,
				expected,
				intent.Dungeon,
			)
		}
	}
}

func TestDungeonV4ParserChaosList(
	t *testing.T,
) {
	for _, input := range []string{
		"!dungeons caos",
		"!dungeons chaos",
	} {

		intent, ok :=
			parseDungeonCommand(
				input,
			)

		if !ok {
			t.Fatalf(
				"%q deveria ser reconhecido",
				input,
			)
		}

		if intent.Action !=
			dungeonActionChaosList {

			t.Fatalf(
				"%q: esperado chaos-list, recebido %q",
				input,
				intent.Action,
			)
		}
	}
}

func TestDungeonV4ParserChaosDungeons(
	t *testing.T,
) {
	tests :=
		map[string]string{
			"!dungeons fenda":        dungeonChaosRift,
			"!dungeons catedral":     dungeonVoidCathedral,
			"!dungeons coracao-fim":  dungeonEndHeart,
			"!dungeons fenda entrar": dungeonChaosRift,
		}

	for input, expected := range tests {

		intent, ok :=
			parseDungeonCommand(
				input,
			)

		if !ok {
			t.Fatalf(
				"%q deveria ser reconhecido",
				input,
			)
		}

		if intent.Dungeon != expected {
			t.Fatalf(
				"%q: esperado %q, recebido %q",
				input,
				expected,
				intent.Dungeon,
			)
		}
	}
}
