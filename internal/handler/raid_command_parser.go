package handler

import "strings"

type raidCommandAction string

const (
	raidActionStatus raidCommandAction = "status"
	raidActionStart  raidCommandAction = "start"
	raidActionJoin   raidCommandAction = "join"
)

func parseRaidCommand(
	text string,
) (
	raidCommandAction,
	bool,
) {
	parts :=
		strings.Fields(
			text,
		)

	if len(parts) == 0 {
		return "",
			false
	}

	command :=
		normalizeCommandToken(
			parts[0],
		)

	switch command {

	case "!boss",
		"!raid",
		"!raide":

	default:

		return "",
			false
	}

	if len(parts) == 1 {
		return raidActionStatus,
			true
	}

	if len(parts) != 2 {
		return "",
			false
	}

	action :=
		normalizeCommandToken(
			parts[1],
		)

	switch action {

	case "status",
		"info",
		"ver":

		return raidActionStatus,
			true

	case "iniciar",
		"abrir",
		"start":

		return raidActionStart,
			true

	case "entrar",
		"participar",
		"join":

		return raidActionJoin,
			true
	}

	return "",
		false
}
