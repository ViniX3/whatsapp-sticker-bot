package handler

import "strings"

func knownCommands() []string {

	known :=
		make(
			map[string]struct{},
		)

	// O menu funciona como catálogo principal.
	for _, field := range strings.Fields(
		menuMessage,
	) {

		token :=
			strings.Trim(
				field,
				"*`.,:;()[]{}",
			)

		if !strings.HasPrefix(
			token,
			"!",
		) {
			continue
		}

		// Segurança para tokens que eventualmente
		// tragam sintaxe grudada.
		for _, separator := range []string{
			"<",
			"|",
		} {

			if index :=
				strings.Index(
					token,
					separator,
				); index > 0 {

				token =
					token[:index]
			}
		}

		command :=
			canonicalCommand(
				token,
			)

		if command != "" {
			known[command] =
				struct{}{}
		}
	}

	// !menu precisa sugerir a si mesmo.
	known["!menu"] =
		struct{}{}

	// Os destinos dos aliases também são comandos válidos,
	// mesmo que algum deles não apareça no menu.
	for _, canonical := range commandAliases {

		known[canonical] =
			struct{}{}
	}

	result :=
		make(
			[]string,
			0,
			len(known),
		)

	for command := range known {

		result =
			append(
				result,
				command,
			)
	}

	return result
}

// suggestCommand procura um comando suficientemente próximo.
//
// IMPORTANTE:
//
// Esta função APENAS sugere.
// Ela nunca deve executar o comando encontrado.
func suggestCommand(
	input string,
) (
	string,
	bool,
) {

	input =
		normalizeCommandToken(
			input,
		)

	if !strings.HasPrefix(
		input,
		"!",
	) {
		return "",
			false
	}

	inputName :=
		strings.TrimPrefix(
			input,
			"!",
		)

	if inputName == "" {
		return "",
			false
	}

	bestCommand := ""
	bestDistance := -1
	bestCount := 0

	for _, candidate := range knownCommands() {

		candidate =
			normalizeCommandToken(
				candidate,
			)

		candidateName :=
			strings.TrimPrefix(
				candidate,
				"!",
			)

		distance :=
			levenshteinDistance(
				inputName,
				candidateName,
			)

		if distance == 0 {
			// É o mesmo comando.
			// Se chegou ao unknown handler,
			// não é um erro de digitação.
			continue
		}

		if bestDistance == -1 ||
			distance < bestDistance {

			bestDistance =
				distance

			bestCommand =
				candidate

			bestCount = 1

			continue
		}

		if distance == bestDistance {
			bestCount++
		}
	}

	if bestCommand == "" ||
		bestDistance < 0 {

		return "",
			false
	}

	maxDistance :=
		maxCommandSuggestionDistance(
			len([]rune(inputName)),
		)

	if bestDistance >
		maxDistance {

		return "",
			false
	}

	// Se dois ou mais comandos ficaram empatados,
	// preferimos não adivinhar.
	if bestCount != 1 {
		return "",
			false
	}

	return bestCommand,
		true
}

func maxCommandSuggestionDistance(
	length int,
) int {

	switch {
	case length <= 4:
		return 1

	case length <= 8:
		return 2

	default:
		return 3
	}
}

func levenshteinDistance(
	left string,
	right string,
) int {

	a :=
		[]rune(left)

	b :=
		[]rune(right)

	if len(a) == 0 {
		return len(b)
	}

	if len(b) == 0 {
		return len(a)
	}

	previous :=
		make(
			[]int,
			len(b)+1,
		)

	current :=
		make(
			[]int,
			len(b)+1,
		)

	for j := 0; j <= len(b); j++ {

		previous[j] =
			j
	}

	for i := 1; i <= len(a); i++ {

		current[0] =
			i

		for j := 1; j <= len(b); j++ {

			cost := 0

			if a[i-1] !=
				b[j-1] {

				cost = 1
			}

			deletion :=
				previous[j] + 1

			insertion :=
				current[j-1] + 1

			substitution :=
				previous[j-1] +
					cost

			current[j] =
				minCommandDistance(
					deletion,
					insertion,
					substitution,
				)
		}

		previous,
			current =
			current,
			previous
	}

	return previous[len(b)]
}

func minCommandDistance(
	a int,
	b int,
	c int,
) int {

	if a <= b &&
		a <= c {

		return a
	}

	if b <= a &&
		b <= c {

		return b
	}

	return c
}
