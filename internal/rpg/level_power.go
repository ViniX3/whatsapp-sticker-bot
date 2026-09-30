package rpg

import (
	"database/sql"
	"fmt"

	"whatsapp-sticker-bot/internal/database"
	"whatsapp-sticker-bot/internal/profile"
)

// LevelCombatPowerBonus retorna o PC permanente
// adquirido através do nível.
//
// A progressão é cumulativa e utiliza faixas:
//
//	Níveis   1-100  = +10 PC por nível
//	Níveis 101-250  = +20 PC por nível
//	Níveis 251-400  = +25 PC por nível
//	Níveis 401-600  = +30 PC por nível
//	Níveis 601-750  = +35 PC por nível
//	Níveis 751-850  = +40 PC por nível
//	Níveis 851-950  = +45 PC por nível
//	Níveis 951-1000 = +50 PC por nível
//
// O nível 1 é o ponto inicial e não concede bônus.
//
// Checkpoints:
//
//	Nível 1    =      0 PC
//	Nível 100  =    990 PC
//	Nível 250  =  3.990 PC
//	Nível 400  =  7.740 PC
//	Nível 600  = 13.740 PC
//	Nível 750  = 18.990 PC
//	Nível 850  = 22.990 PC
//	Nível 950  = 27.490 PC
//	Nível 1000 = 29.990 PC
func LevelCombatPowerBonus(
	level int,
) int {

	if level <= 1 {
		return 0
	}

	// Não existe progressão oficial acima
	// do nível máximo atual.
	if level > 1000 {
		level = 1000
	}

	type levelPowerBand struct {
		start int
		end   int
		power int
	}

	bands := [...]levelPowerBand{
		{start: 2, end: 100, power: 10},
		{start: 101, end: 250, power: 20},
		{start: 251, end: 400, power: 25},
		{start: 401, end: 600, power: 30},
		{start: 601, end: 750, power: 35},
		{start: 751, end: 850, power: 40},
		{start: 851, end: 950, power: 45},
		{start: 951, end: 1000, power: 50},
	}

	total := 0

	for _, band := range bands {

		if level < band.start {
			break
		}

		lastLevel := level

		if lastLevel > band.end {
			lastLevel = band.end
		}

		levelsInBand :=
			lastLevel - band.start + 1

		total +=
			levelsInBand * band.power
	}

	return total
}

// playerLevelPowerBonus consulta o XP do jogador,
// calcula seu nível pelo sistema central de progressão
// e converte esse nível em Poder de Combate.
func playerLevelPowerBonus(
	groupJID string,
	jid string,
) (
	int,
	int,
	error,
) {

	if database.DB == nil {
		return 1,
			0,
			fmt.Errorf(
				"banco de dados não inicializado",
			)
	}

	var xp int

	err :=
		database.DB.QueryRow(`
			SELECT xp
			FROM player_profiles
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&xp,
		)

	if err == sql.ErrNoRows {
		return 1, 0, nil
	}

	if err != nil {
		return 1,
			0,
			fmt.Errorf(
				"erro consultando XP do jogador: %w",
				err,
			)
	}

	level :=
		profile.LevelFromXP(
			xp,
		).Level

	return level,
		LevelCombatPowerBonus(
			level,
		),
		nil
}
