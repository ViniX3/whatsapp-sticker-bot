package handler

import (
	"testing"

	"whatsapp-sticker-bot/internal/rpg"
)

func TestRPGDungeonVictoryXP(
	t *testing.T,
) {

	tests := []struct {
		id   string
		want int
	}{
		{"ruinas", 150},
		{"cripta", 350},
		{"fortaleza", 550},
	}

	for _, test := range tests {

		result :=
			&rpg.DungeonAttemptResult{
				Won: true,
				Dungeon: rpg.Dungeon{
					ID: test.id,
				},
			}

		got :=
			rpgDungeonVictoryXP(
				result,
			)

		if got != test.want {

			t.Fatalf(
				"%s: esperado %d XP, recebido %d",
				test.id,
				test.want,
				got,
			)
		}
	}
}

func TestRPGDungeonXPByTier(
	t *testing.T,
) {

	tests := []struct {
		tier int
		want int
	}{
		{0, 150},
		{1, 350},
		{2, 550},
		{3, 750},
		{4, 950},
	}

	for _, test := range tests {

		got :=
			rpgDungeonXPByTier(
				test.tier,
			)

		if got != test.want {

			t.Fatalf(
				"tier %d: esperado %d XP, recebido %d",
				test.tier,
				test.want,
				got,
			)
		}
	}
}

func TestRPGBossVictoryXP(
	t *testing.T,
) {

	tests := []struct {
		rarity rpg.Rarity
		want   int
	}{
		{rpg.RarityEpic, 150},
		{rpg.RarityLegendary, 250},
		{rpg.RarityMythic, 400},
		{rpg.RaritySacred, 550},
	}

	for _, test := range tests {

		got :=
			rpgBossVictoryXP(
				test.rarity,
			)

		if got != test.want {

			t.Fatalf(
				"raridade %s: esperado %d XP, recebido %d",
				test.rarity,
				test.want,
				got,
			)
		}
	}
}

func TestRPGPVERegularVictoryXP(
	t *testing.T,
) {

	result :=
		&rpg.PVEResult{
			Won: true,
			Enemy: rpg.Enemy{
				Rarity: rpg.RarityLegendary,
			},
		}

	got :=
		rpgPVEVictoryXP(
			result,
		)

	if got != 100 {
		t.Fatalf(
			"lendário comum: esperado 100 XP, recebido %d",
			got,
		)
	}
}
