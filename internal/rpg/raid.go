package rpg

import "time"

const (
	RaidMaxParticipants = 10
	RaidJoinWindow      = 90 * time.Second

	// Enquanto ninguém entrar, o Boss permanece no mundo.
	// O starts_at distante impede o worker de resolver a Raid.
	RaidDormantWindow = 100 * 365 * 24 * time.Hour

	// Intervalo inicial usado durante o teste controlado.
	// Depois podemos balancear vitória/derrota separadamente.
	RaidRespawnDelay = 20 * time.Minute
)

type RaidBonuses struct {
	BossDamagePercent int

	BlessingChancePercent int

	CrystalRewardPercent int

	StellarStoneChancePercent int

	DropChancePercent int

	RareDropChancePercent int

	DropRarityUpgradeChancePercent int
}

type RaidParticipantPower struct {
	CombatPower int

	Bonuses RaidBonuses
}

// RaidBonusesFromSets consolida todos os bônus relevantes
// para Bosses/Raids vindos dos sets ativos do jogador.
func RaidBonusesFromSets(
	active []ActiveSetBonus,
) RaidBonuses {

	var result RaidBonuses

	for _, bonus := range active {

		for _, effect := range bonus.Effects {

			if effect.Value <= 0 {
				continue
			}

			switch effect.Type {

			case BonusBossDamagePercent:
				result.BossDamagePercent +=
					effect.Value

			case BonusBlessingChancePercent:
				result.BlessingChancePercent +=
					effect.Value

			case BonusCrystalRewardPercent:
				result.CrystalRewardPercent +=
					effect.Value

			case BonusStellarStoneChancePercent:
				result.StellarStoneChancePercent +=
					effect.Value

			case BonusDropChancePercent:
				result.DropChancePercent +=
					effect.Value

			case BonusRareDropChancePercent:
				result.RareDropChancePercent +=
					effect.Value

			case BonusDropRarityUpgradeChancePercent:
				result.DropRarityUpgradeChancePercent +=
					effect.Value
			}
		}
	}

	return result
}

// EffectiveBossPower retorna o PC efetivo do jogador
// especificamente contra Bosses.
//
// O CombatPower recebido aqui já representa o PC final
// calculado pelo sistema de equipamentos. Portanto,
// aplicamos somente BOSS_DAMAGE_PERCENT.
func (
	participant RaidParticipantPower,
) EffectiveBossPower() int {

	return RaidEffectiveBossPower(
		participant.CombatPower,
		participant.Bonuses.
			BossDamagePercent,
	)
}

func RaidEffectiveBossPower(
	combatPower int,
	bossDamagePercent int,
) int {

	if combatPower <= 0 {
		return 0
	}

	if bossDamagePercent <= 0 {
		return combatPower
	}

	value :=
		int64(combatPower) *
			int64(
				100+
					bossDamagePercent,
			)

	// Arredondamento para o inteiro mais próximo.
	value =
		(value + 50) /
			100

	if value <= 0 {
		return 0
	}

	return int(value)
}

// RaidTotalPower soma o poder efetivo contra Bosses
// de todos os participantes.
func RaidTotalPower(
	participants []RaidParticipantPower,
) int {

	total :=
		int64(0)

	for _, participant := range participants {

		total +=
			int64(
				participant.
					EffectiveBossPower(),
			)
	}

	if total <= 0 {
		return 0
	}

	return int(total)
}

// RaidSuccessChance calcula a chance percentual da RAID.
//
// Fórmula:
//
//	RAID / (RAID + BOSS)
//
// Exemplos:
//
//	mesmo poder       -> 50%
//	RAID 2x superior -> 67%
//	RAID 3x superior -> 75%
func RaidSuccessChance(
	raidPower int,
	bossPower int,
) int {

	if raidPower <= 0 ||
		bossPower <= 0 {

		return 0
	}

	numerator :=
		int64(raidPower) *
			100

	denominator :=
		int64(raidPower) +
			int64(bossPower)

	chance :=
		int(
			(numerator +
				denominator/2) /
				denominator,
		)

	if chance < 1 {
		return 1
	}

	if chance > 99 {
		return 99
	}

	return chance
}
