package handler

import (
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/profile"
	"whatsapp-sticker-bot/internal/rpg"
)

const (
	RPGGatheringXPReward = 25

	// Dungeons começam em 150 XP.
	// Cada novo tier adiciona +200 XP.
	RPGDungeonBaseXP = 150
	RPGDungeonXPStep = 200
)

func recordRPGXP(
	groupJID string,
	jid string,
	amount int,
) string {

	if amount <= 0 {
		return ""
	}

	result, err :=
		profile.AddXP(
			groupJID,
			jid,
			amount,
		)

	if err != nil {
		logger.Error(
			"Erro ao registrar XP RPG:",
			err,
		)

		return ""
	}

	return progressionSuffix(
		result,
	)
}

func rpgPVEVictoryXP(
	result *rpg.PVEResult,
) int {

	if result == nil ||
		!result.Won {

		return 0
	}

	// Boss possui progressão própria de XP
	// baseada na raridade.
	if result.BossKill {
		return rpgBossVictoryXP(
			result.Enemy.Rarity,
		)
	}

	switch result.Enemy.Rarity {

	case rpg.RarityCommon:
		return 10

	case rpg.RarityRare:
		return 25

	case rpg.RarityEpic:
		return 60

	case rpg.RarityLegendary:
		return 100

	case rpg.RarityMythic:
		return 150

	case rpg.RaritySacred:
		return 225

	default:
		return 10
	}
}

func rpgBossVictoryXP(
	rarity rpg.Rarity,
) int {

	switch rarity {

	case rpg.RarityWorn:
		return 50

	case rpg.RarityCommon:
		return 75

	case rpg.RarityRare:
		return 100

	case rpg.RarityEpic:
		return 150

	case rpg.RarityLegendary:
		return 250

	case rpg.RarityMythic:
		return 400

	case rpg.RaritySacred:
		return 550

	default:
		return 150
	}
}

func rpgDungeonXPByTier(
	tier int,
) int {

	if tier < 0 {
		return 0
	}

	return RPGDungeonBaseXP +
		(tier * RPGDungeonXPStep)
}

func rpgDungeonVictoryXP(
	result *rpg.DungeonAttemptResult,
) int {

	if result == nil ||
		!result.Won {

		return 0
	}

	switch result.Dungeon.ID {

	case "ruinas":
		return rpgDungeonXPByTier(0)

	case "cripta":
		return rpgDungeonXPByTier(1)

	case "fortaleza":
		return rpgDungeonXPByTier(2)

	default:
		return RPGDungeonBaseXP
	}
}
