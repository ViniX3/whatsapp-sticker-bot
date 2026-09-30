package handler

import (
	"strings"
	"unicode"
)

// commandAliases contém aliases que podem ser transformados
// diretamente no comando canônico.
//
// Exemplo:
//
//	!aposta 20%
//	→ !bet 20%
var commandAliases = map[string]string{
	"!inv":        "!inventario",
	"!inventario": "!inventario",

	"!dungeon":   "!dungeons",
	"!dungeons":  "!dungeons",
	"!masmorra":  "!dungeons",
	"!masmorras": "!dungeons",

	// Alias contextual.
	//
	// Ele é utilizado para roteamento pelo dispatcher,
	// mas normalizeCommandText preserva a palavra !entrar,
	// pois:
	//
	//	!entrar fortaleza
	//
	// possui significado diferente de:
	//
	//	!dungeons fortaleza
	"!entrar": "!dungeons",

	"!bet":     "!bet",
	"!aposta":  "!bet",
	"!apostar": "!bet",

	"!cristais": "!cristais",

	// Loja
	"!comprar":  "!comprar",
	"!adquirir": "!comprar",

	// Forja
	"!forjar":   "!forjar",
	"!fabricar": "!forjar",
	"!criar":    "!forjar",
	"!produzir": "!forjar",
	"!craft":    "!forjar",

	// PvE
	"!pve":      "!pve",
	"!cacar":    "!pve",
	"!lutar":    "!pve",
	"!combater": "!pve",
	// Equipamentos
	"!equip":   "!equipar",
	"!equipar": "!equipar",
	"!vestir":  "!equipar",

	"!desequipar": "!desequipar",
	"!tirar":      "!desequipar",
	"!desvestir":  "!desequipar",

	"!equipamentos": "!equipamentos",
	"!equipamento":  "!equipamentos",
	"!gear":         "!equipamentos",

	// Consultas / navegação
	"!mochila":    "!inventario",
	"!bolsa":      "!inventario",
	"!arsenal":    "!equipamentos",
	"!ficha":      "!perfil",
	"!personagem": "!perfil",
}

// contextualCommandAliases não são reescritos dentro do texto.
//
// Eles ainda são resolvidos por canonicalCommand() para que
// o dispatcher saiba qual handler deve recebê-los.
var contextualCommandAliases = map[string]struct{}{
	"!entrar": {},
}

// normalizeCommandToken cria uma representação interna
// previsível para palavras de controle.
//
// Exemplos:
//
//	RUÍNAS      -> ruinas
//	Fortaleza   -> fortaleza
//	!Inventário -> !inventario
func normalizeCommandToken(
	value string,
) string {

	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	replacer :=
		strings.NewReplacer(
			"á", "a",
			"à", "a",
			"â", "a",
			"ã", "a",
			"ä", "a",

			"é", "e",
			"è", "e",
			"ê", "e",
			"ë", "e",

			"í", "i",
			"ì", "i",
			"î", "i",
			"ï", "i",

			"ó", "o",
			"ò", "o",
			"ô", "o",
			"õ", "o",
			"ö", "o",

			"ú", "u",
			"ù", "u",
			"û", "u",
			"ü", "u",

			"ç", "c",
		)

	return replacer.Replace(value)
}

// canonicalCommand resolve um comando ou alias para sua
// representação oficial.
func canonicalCommand(
	value string,
) string {

	command :=
		normalizeCommandToken(
			value,
		)

	if canonical,
		ok :=
		commandAliases[command]; ok {

		return canonical
	}

	return command
}

// normalizeCommandText normaliza SOMENTE o primeiro token
// da mensagem.
//
// Os argumentos permanecem intactos.
//
// Exemplos:
//
//	!INVENTÁRIO 2
//	→ !inventario 2
//
//	!APOSTA 20%
//	→ !bet 20%
//
//	!Entrar Ruínas
//	→ !entrar Ruínas
//
// A última forma é proposital: !entrar é contextual e precisa
// chegar assim ao parser de Dungeons.
func normalizeCommandText(
	text string,
) string {

	if strings.TrimSpace(text) == "" {
		return text
	}

	leftTrimmed :=
		strings.TrimLeftFunc(
			text,
			unicode.IsSpace,
		)

	prefixLength :=
		len(text) -
			len(leftTrimmed)

	prefix :=
		text[:prefixLength]

	commandEnd :=
		strings.IndexFunc(
			leftTrimmed,
			unicode.IsSpace,
		)

	var (
		rawCommand string
		suffix     string
	)

	if commandEnd == -1 {
		rawCommand =
			leftTrimmed
	} else {
		rawCommand =
			leftTrimmed[:commandEnd]

		suffix =
			leftTrimmed[commandEnd:]
	}

	normalized :=
		normalizeCommandToken(
			rawCommand,
		)

	if _, contextual :=
		contextualCommandAliases[normalized]; contextual {

		return prefix +
			normalized +
			suffix
	}

	return prefix +
		canonicalCommand(
			normalized,
		) +
		suffix
}
