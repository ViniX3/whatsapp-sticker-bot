package handler

import (
	"errors"
	"strconv"
	"strings"
)

var (
	errFlexibleAmountInvalid = errors.New(
		"valor inválido",
	)

	errFlexibleAmountPercentRange = errors.New(
		"percentual precisa estar entre 1 e 100",
	)

	errFlexibleAmountNoBalance = errors.New(
		"saldo insuficiente para valor relativo",
	)
)

// parseFlexibleGoldAmount interpreta valores financeiros.
//
// Formatos aceitos:
//
//	1000
//	1.000
//	1_000
//	all
//	tudo
//	max
//	maximo
//	saldo
//	metade
//	half
//	20%
//	50%
//	100%
//
// balance só é utilizado quando o argumento depende
// do saldo atual.
func parseFlexibleGoldAmount(
	value string,
	balance int,
) (
	int,
	error,
) {

	value =
		normalizeCommandToken(
			value,
		)

	switch value {

	case "all",
		"tudo",
		"todos",
		"max",
		"maximo",
		"saldo":

		if balance <= 0 {
			return 0,
				errFlexibleAmountNoBalance
		}

		return balance,
			nil

	case "metade",
		"half":

		if balance <= 0 {
			return 0,
				errFlexibleAmountNoBalance
		}

		amount :=
			balance / 2

		if amount <= 0 {
			return 0,
				errFlexibleAmountNoBalance
		}

		return amount,
			nil
	}

	// ========================================================
	// PERCENTUAL
	// ========================================================

	if strings.HasSuffix(
		value,
		"%",
	) {

		rawPercent :=
			strings.TrimSuffix(
				value,
				"%",
			)

		percent,
			err :=
			strconv.Atoi(
				rawPercent,
			)

		if err != nil {
			return 0,
				errFlexibleAmountInvalid
		}

		if percent < 1 ||
			percent > 100 {

			return 0,
				errFlexibleAmountPercentRange
		}

		if balance <= 0 {
			return 0,
				errFlexibleAmountNoBalance
		}

		// Calcula balance * percent / 100
		// sem gerar multiplicação intermediária perigosa.
		//
		// Exemplo:
		//
		// balance = 10^18
		// percent = 50
		//
		// Evitamos:
		//
		// balance * 50
		//
		// dividindo antes.
		whole :=
			balance / 100

		remainder :=
			balance % 100

		amount :=
			whole*percent +
				(remainder*percent)/100

		if amount <= 0 {
			return 0,
				errFlexibleAmountNoBalance
		}

		return amount,
			nil
	}

	// ========================================================
	// VALOR ABSOLUTO
	// ========================================================

	// Aceita:
	//
	// 1000000
	// 1.000.000
	// 1_000_000
	clean :=
		strings.NewReplacer(
			".", "",
			"_", "",
		).Replace(
			value,
		)

	amount,
		err :=
		strconv.Atoi(
			clean,
		)

	if err != nil ||
		amount <= 0 {

		return 0,
			errFlexibleAmountInvalid
	}

	return amount,
		nil
}
