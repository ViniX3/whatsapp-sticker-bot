package rpg

const (
	MinCombatSuccessChance = 10
	MaxCombatSuccessChance = 90
)

// CombatComparison representa uma comparação direta
// de Poder de Combate entre dois jogadores.
type CombatComparison struct {
	AttackerPower int
	DefenderPower int

	SuccessChance int
}

// CompareCombatPower calcula a chance percentual do atacante.
//
// Fórmula base:
//
//	PC atacante / (PC atacante + PC defensor) * 100
//
// O resultado final sempre fica entre 10% e 90%.
//
// Regras:
//
//   - poderes negativos são tratados como 0;
//   - 0 PC contra 0 PC resulta em 50%;
//   - ninguém possui vitória garantida;
//   - ninguém fica completamente imune.
func CompareCombatPower(
	attackerPower int,
	defenderPower int,
) CombatComparison {
	if attackerPower < 0 {
		attackerPower = 0
	}

	if defenderPower < 0 {
		defenderPower = 0
	}

	chance := 50

	total := attackerPower + defenderPower

	if total > 0 {
		chance = int((int64(attackerPower) * 100) / int64(total))
	}

	if chance < MinCombatSuccessChance {
		chance = MinCombatSuccessChance
	}

	if chance > MaxCombatSuccessChance {
		chance = MaxCombatSuccessChance
	}

	return CombatComparison{
		AttackerPower: attackerPower,
		DefenderPower: defenderPower,
		SuccessChance: chance,
	}
}

// CombatSuccessChance retorna apenas a chance percentual
// final da comparação.
func CombatSuccessChance(
	attackerPower int,
	defenderPower int,
) int {
	return CompareCombatPower(
		attackerPower,
		defenderPower,
	).SuccessChance
}
