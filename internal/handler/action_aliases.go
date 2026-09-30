package handler

const (
	actionDeposit  = "deposit"
	actionWithdraw = "withdraw"
	actionTypes    = "types"
	actionBuy      = "buy"
	actionCraft    = "craft"
)

// canonicalAction transforma palavras naturais em uma ação
// interna previsível.
//
// Essa camada será reaproveitada futuramente por outros
// sistemas como Loja e Forja.
func canonicalAction(
	value string,
) (
	string,
	bool,
) {

	switch normalizeCommandToken(
		value,
	) {

	// ========================================================
	// DEPOSITAR / GUARDAR
	// ========================================================

	case "guardar",
		"guarda",
		"depositar",
		"deposito",
		"colocar",
		"armazenar":

		return actionDeposit,
			true

	// ========================================================
	// RETIRAR / SACAR
	// ========================================================

	case "retirar",
		"retira",
		"sacar",
		"saque",
		"pegar",
		"remover":

		return actionWithdraw,
			true

	// ========================================================
	// LISTAGEM / TIPOS
	// ========================================================

	// ========================================================
	// COMPRAR / ADQUIRIR
	// ========================================================

	case "comprar",
		"compra",
		"adquirir",
		"adquire":

		return actionBuy,
			true

	// ========================================================
	// FORJAR / FABRICAR
	// ========================================================

	case "forjar",
		"fabricar",
		"criar",
		"produzir",
		"craft":

		return actionCraft,
			true

	// ========================================================
	// LISTAGEM / TIPOS
	// ========================================================

	case "tipo",
		"tipos",
		"cofres",
		"niveis":

		return actionTypes,
			true
	}

	return "",
		false
}
