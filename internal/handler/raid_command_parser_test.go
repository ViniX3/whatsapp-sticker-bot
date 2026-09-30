package handler

import "testing"

func TestParseRaidCommand(
	t *testing.T,
) {
	tests :=
		map[string]raidCommandAction{
			"!boss": raidActionStatus,

			"!boss status": raidActionStatus,

			"!boss info": raidActionStatus,

			"!boss iniciar": raidActionStart,

			"!boss abrir": raidActionStart,

			"!boss entrar": raidActionJoin,

			"!boss participar": raidActionJoin,

			"!raid": raidActionStatus,

			"!raid entrar": raidActionJoin,

			"!raide iniciar": raidActionStart,
		}

	for input, expected := range tests {

		action, ok :=
			parseRaidCommand(
				input,
			)

		if !ok {
			t.Fatalf(
				"%q deveria ser aceito",
				input,
			)
		}

		if action != expected {
			t.Fatalf(
				"%q: esperado %s, recebido %s",
				input,
				expected,
				action,
			)
		}
	}
}

func TestParseRaidCommandRejectsInvalid(
	t *testing.T,
) {
	tests :=
		[]string{
			"",
			"!boss qualquercoisa",
			"!boss entrar agora",
			"!raid desconhecido",
			"!pve",
		}

	for _, input := range tests {

		_, ok :=
			parseRaidCommand(
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
