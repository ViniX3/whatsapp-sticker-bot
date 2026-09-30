package handler

import (
	"strconv"
	"strings"
)

func formatTotalValue(
	value int,
	resource string,
) string {

	exact :=
		formatTotalExact(
			int64(value),
		)

	scale := ""

	switch {
	case value >= 1_000_000_000_000_000_000:
		scale = "quintilhão"

	case value >= 1_000_000_000_000_000:
		scale = "quatrilhão"

	case value >= 1_000_000_000_000:
		scale = "trilhão"

	case value >= 1_000_000_000:
		scale = "bilhão"

	case value >= 1_000_000:
		scale = "milhão"

	case value >= 1_000:
		scale = "mil"
	}

	if scale == "" {
		return exact +
			" " +
			resource
	}

	return exact +
		" " +
		scale +
		" de " +
		resource
}

func formatTotalExact(
	value int64,
) string {

	negative := value < 0

	if negative {
		value = -value
	}

	text :=
		strconv.FormatInt(
			value,
			10,
		)

	if len(text) <= 3 {
		if negative {
			return "-" + text
		}

		return text
	}

	var builder strings.Builder

	first :=
		len(text) % 3

	if first == 0 {
		first = 3
	}

	builder.WriteString(
		text[:first],
	)

	for position := first; position < len(text); position += 3 {

		builder.WriteByte('.')

		builder.WriteString(
			text[position : position+3],
		)
	}

	if negative {
		return "-" +
			builder.String()
	}

	return builder.String()
}
