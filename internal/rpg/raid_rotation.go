package rpg

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

const (
	RaidLegendaryRotationWeight = 60

	RaidMythicRotationWeight = 35

	RaidSacredRotationWeight = 5

	RaidLegendaryRecentBlock = 6

	RaidMythicRecentBlock = 5

	RaidSacredRecentBlock = 2
)

var ErrRaidRotationNoCandidates = errors.New(
	"nenhum Raid Boss disponível para rotação",
)

func RaidRotationRarityByRoll(
	roll int,
) (
	Rarity,
	error,
) {

	if roll < 0 ||
		roll >= 100 {

		return "",
			fmt.Errorf(
				"roll de rotação inválido: %d",
				roll,
			)
	}

	legendaryLimit :=
		RaidLegendaryRotationWeight

	mythicLimit :=
		legendaryLimit +
			RaidMythicRotationWeight

	switch {

	case roll < legendaryLimit:

		return RarityLegendary,
			nil

	case roll < mythicLimit:

		return RarityMythic,
			nil

	default:

		return RaritySacred,
			nil
	}
}

func RaidRotationRecentLimit(
	rarity Rarity,
) int {

	switch rarity {

	case RarityLegendary:

		return RaidLegendaryRecentBlock

	case RarityMythic:

		return RaidMythicRecentBlock

	case RaritySacred:

		return RaidSacredRecentBlock
	}

	return 0
}

func RaidRotationCandidates(
	catalog *RaidBossCatalog,
	rarity Rarity,
	recentBossIDs []string,
) []RaidBoss {

	if catalog == nil {
		return nil
	}

	allBosses :=
		catalog.BossesByRarity(
			rarity,
		)

	bosses :=
		make(
			[]RaidBoss,
			0,
			len(allBosses),
		)

	for _, boss := range allBosses {
		if !RaidBossEnabledForProgression(
			boss,
		) {
			continue
		}

		bosses =
			append(
				bosses,
				boss,
			)
	}

	if len(bosses) == 0 {
		return nil
	}

	limit :=
		RaidRotationRecentLimit(
			rarity,
		)

	if limit <= 0 ||
		len(recentBossIDs) == 0 {

		return bosses
	}

	blocked :=
		make(
			map[string]struct{},
			limit,
		)

	for index :=
		len(recentBossIDs) - 1; index >= 0 &&
		len(blocked) < limit; index-- {

		bossID :=
			strings.ToLower(
				strings.TrimSpace(
					recentBossIDs[index],
				),
			)

		if bossID == "" {
			continue
		}

		boss, exists :=
			catalog.BossByID(
				bossID,
			)

		if !exists ||
			boss.Rarity != rarity {

			continue
		}

		blocked[boss.ID] =
			struct{}{}
	}

	candidates :=
		make(
			[]RaidBoss,
			0,
			len(bosses),
		)

	for _, boss := range bosses {

		if _, exists :=
			blocked[boss.ID]; exists {

			continue
		}

		candidates =
			append(
				candidates,
				boss,
			)
	}

	// Segurança para futuras alterações de catálogo.
	// Caso todo o pool fique bloqueado, libera novamente
	// a raridade em vez de impedir a criação da Raid.
	if len(candidates) == 0 {
		return bosses
	}

	return candidates
}

func SelectRaidBoss(
	catalog *RaidBossCatalog,
	recentBossIDs []string,
) (
	RaidBoss,
	error,
) {

	if catalog == nil {
		return RaidBoss{},
			ErrRaidRotationNoCandidates
	}

	rollValue, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(100),
		)

	if err != nil {
		return RaidBoss{},
			fmt.Errorf(
				"erro sorteando raridade da Raid: %w",
				err,
			)
	}

	rarity, err :=
		RaidRotationRarityByRoll(
			int(
				rollValue.Int64(),
			),
		)

	if err != nil {
		return RaidBoss{},
			err
	}

	candidates :=
		RaidRotationCandidates(
			catalog,
			rarity,
			recentBossIDs,
		)

	if len(candidates) == 0 {
		return RaidBoss{},
			ErrRaidRotationNoCandidates
	}

	indexValue, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(
				int64(
					len(candidates),
				),
			),
		)

	if err != nil {
		return RaidBoss{},
			fmt.Errorf(
				"erro sorteando Raid Boss: %w",
				err,
			)
	}

	return candidates[int(
			indexValue.Int64(),
		)],
		nil
}
