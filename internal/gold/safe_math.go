package gold

import (
	"errors"
	"math/big"
)

var ErrGoldLimitExceeded = errors.New(
	"resultado financeiro excede o limite máximo de Gold",
)

// safeGoldPercent calcula:
//
//	amount * percent / 100
//
// utilizando big.Int internamente.
//
// Isso evita overflow intermediário em operações como:
//
//	2.000.000.000.000.000 * 10.000 / 100
//
// cujo resultado final cabe em int64, mas cuja
// multiplicação intermediária não cabe.
func safeGoldPercent(
	amount int,
	percent int,
) (
	int,
	error,
) {

	if amount < 0 ||
		percent < 0 {

		return 0,
			ErrGoldLimitExceeded
	}

	value :=
		big.NewInt(
			int64(amount),
		)

	percentage :=
		big.NewInt(
			int64(percent),
		)

	result :=
		new(big.Int).Mul(
			value,
			percentage,
		)

	result.Quo(
		result,
		big.NewInt(100),
	)

	if !result.IsInt64() {
		return 0,
			ErrGoldLimitExceeded
	}

	return int(
			result.Int64(),
		),
		nil
}

// safeGoldAdd soma dois valores sem permitir
// overflow do inteiro utilizado pela economia.
func safeGoldAdd(
	left int,
	right int,
) (
	int,
	error,
) {

	result :=
		new(big.Int).Add(
			big.NewInt(
				int64(left),
			),
			big.NewInt(
				int64(right),
			),
		)

	if !result.IsInt64() {
		return 0,
			ErrGoldLimitExceeded
	}

	return int(
			result.Int64(),
		),
		nil
}
