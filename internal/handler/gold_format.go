package handler

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type goldScale struct {
	Value    float64
	Singular string
	Plural   string
}

var goldScales = []goldScale{
	{
		Value:    1_000_000_000_000_000_000,
		Singular: "quintilhão",
		Plural:   "quintilhões",
	},
	{
		Value:    1_000_000_000_000_000,
		Singular: "quatrilhão",
		Plural:   "quatrilhões",
	},
	{
		Value:    1_000_000_000_000,
		Singular: "trilhão",
		Plural:   "trilhões",
	},
	{
		Value:    1_000_000_000,
		Singular: "bilhão",
		Plural:   "bilhões",
	},
	{
		Value:    1_000_000,
		Singular: "milhão",
		Plural:   "milhões",
	},
}

func formatGoldAmount(
	value int,
) string {
	negative := value < 0

	absolute :=
		math.Abs(
			float64(value),
		)

	var result string

	switch {
	case absolute < 1000:
		result =
			formatGoldExact(
				int64(absolute),
			)

	case absolute < 1_000_000:
		result =
			formatGoldScaled(
				absolute/1000,
				"mil",
				"mil",
			)

	default:
		for _, scale := range goldScales {

			if absolute <
				scale.Value {

				continue
			}

			result =
				formatGoldScaled(
					absolute/
						scale.Value,
					scale.Singular,
					scale.Plural,
				)

			break
		}
	}

	if result == "" {
		result =
			formatGoldExact(
				int64(absolute),
			)
	}

	if negative {
		return "-" + result
	}

	return result
}

func formatGold(
	value int,
) string {
	return formatGoldAmount(
		value,
	) + " Gold"
}

func formatSignedGold(
	value int,
) string {
	if value > 0 {
		return "+" +
			formatGoldAmount(value) +
			" Gold"
	}

	return formatGold(value)
}

func formatGoldScaled(
	value float64,
	singular string,
	plural string,
) string {
	text :=
		formatGoldDecimal(
			value,
		)

	unit := plural

	if math.Abs(
		value-1,
	) < 0.000001 {

		unit = singular
	}

	return text + " " + unit
}

func formatGoldDecimal(
	value float64,
) string {
	var format string

	switch {
	case value >= 100:
		format = "%.1f"

	case value >= 10:
		format = "%.1f"

	default:
		format = "%.2f"
	}

	text :=
		fmt.Sprintf(
			format,
			value,
		)

	text =
		strings.TrimRight(
			text,
			"0",
		)

	text =
		strings.TrimRight(
			text,
			".",
		)

	text =
		strings.ReplaceAll(
			text,
			".",
			",",
		)

	return text
}

func formatGoldExact(
	value int64,
) string {
	text :=
		strconv.FormatInt(
			value,
			10,
		)

	if len(text) <= 3 {
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

	for i := first; i < len(text); i += 3 {

		builder.WriteString(".")
		builder.WriteString(
			text[i : i+3],
		)
	}

	return builder.String()
}
