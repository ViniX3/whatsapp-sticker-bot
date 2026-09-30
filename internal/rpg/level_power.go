package rpg

import (
	"database/sql"
	"fmt"

	"whatsapp-sticker-bot/internal/database"
	"whatsapp-sticker-bot/internal/profile"
)

const CombatPowerPerLevel = 10

// LevelCombatPowerBonus retorna o PC permanente
// adquirido através do nível.
//
// Nível 1   = +0 PC
// Nível 2   = +10 PC
// Nível 10  = +90 PC
// Nível 100 = +990 PC
// Nível 200 = +1990 PC
func LevelCombatPowerBonus(
	level int,
) int {

	if level <= 1 {
		return 0
	}

	return (level - 1) *
		CombatPowerPerLevel
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
