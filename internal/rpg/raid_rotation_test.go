package rpg

import "testing"

func TestRaidRotationWeights(
	t *testing.T,
) {

	total :=
		RaidLegendaryRotationWeight +
			RaidMythicRotationWeight +
			RaidSacredRotationWeight

	if total != 100 {
		t.Fatalf(
			"peso total esperado 100, recebido %d",
			total,
		)
	}
}

func TestRaidRotationRarityByRoll(
	t *testing.T,
) {

	tests :=
		[]struct {
			roll int
			want Rarity
		}{
			{
				roll: 0,
				want: RarityLegendary,
			},
			{
				roll: 59,
				want: RarityLegendary,
			},
			{
				roll: 60,
				want: RarityMythic,
			},
			{
				roll: 94,
				want: RarityMythic,
			},
			{
				roll: 95,
				want: RaritySacred,
			},
			{
				roll: 99,
				want: RaritySacred,
			},
		}

	for _, test := range tests {

		got, err :=
			RaidRotationRarityByRoll(
				test.roll,
			)

		if err != nil {
			t.Fatal(err)
		}

		if got != test.want {
			t.Fatalf(
				"roll %d: esperado %s, recebido %s",
				test.roll,
				test.want,
				got,
			)
		}
	}
}

func TestRaidRotationRejectsInvalidRoll(
	t *testing.T,
) {

	for _, roll := range []int{
		-1,
		100,
		150,
	} {

		_, err :=
			RaidRotationRarityByRoll(
				roll,
			)

		if err == nil {
			t.Fatalf(
				"roll %d deveria falhar",
				roll,
			)
		}
	}
}

func TestRaidRotationCandidateCounts(
	t *testing.T,
) {

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	tests :=
		[]struct {
			rarity Rarity
			want   int
		}{
			{
				rarity: RarityLegendary,

				want: 2,
			},
			{
				rarity: RarityMythic,

				want: 15,
			},
			{
				rarity: RaritySacred,

				want: 3,
			},
		}

	for _, test := range tests {

		candidates :=
			RaidRotationCandidates(
				catalog,
				test.rarity,
				nil,
			)

		if len(candidates) !=
			test.want {

			t.Fatalf(
				"%s: esperado %d candidatos, recebido %d",
				test.rarity,
				test.want,
				len(candidates),
			)
		}
	}
}

func TestRaidRotationBlocksRecentLegendary(
	t *testing.T,
) {

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	// Apenas Crimson Chimera está recente.
	// Ash Cerberus deve ser o único candidato.
	recent :=
		[]string{
			"crimson_chimera",
		}

	candidates :=
		RaidRotationCandidates(
			catalog,
			RarityLegendary,
			recent,
		)

	if len(candidates) != 1 {
		t.Fatalf(
			"esperado 1 Legendary disponível, recebidos %d",
			len(candidates),
		)
	}

	if candidates[0].ID !=
		"ash_cerberus" {

		t.Fatalf(
			"esperado ash_cerberus, recebido %s",
			candidates[0].ID,
		)
	}

	// Se os dois Legendary ativos estiverem recentes,
	// o fallback deve liberar novamente o pool completo
	// para não impedir a criação de uma Raid Legendary.
	recent =
		[]string{
			"crimson_chimera",
			"ash_cerberus",
		}

	candidates =
		RaidRotationCandidates(
			catalog,
			RarityLegendary,
			recent,
		)

	if len(candidates) != 2 {
		t.Fatalf(
			"fallback Legendary deveria restaurar 2 candidatos; recebeu %d",
			len(candidates),
		)
	}
}

func TestRaidRotationBlocksRecentMythic(
	t *testing.T,
) {

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	recent :=
		[]string{
			"phoenix",
			"ancient_titan",
			"world_serpent",
			"celestial_hydra",
			"abyss_king",
		}

	candidates :=
		RaidRotationCandidates(
			catalog,
			RarityMythic,
			recent,
		)

	if len(candidates) != 10 {
		t.Fatalf(
			"esperados 10 Mythic disponíveis, recebidos %d",
			len(candidates),
		)
	}
}

func TestRaidRotationBlocksRecentSacred(
	t *testing.T,
) {

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	recent :=
		[]string{
			"bahamut",
			"azathor",
		}

	candidates :=
		RaidRotationCandidates(
			catalog,
			RaritySacred,
			recent,
		)

	if len(candidates) != 1 {
		t.Fatalf(
			"esperado 1 Sacred disponível, recebidos %d",
			len(candidates),
		)
	}

	if candidates[0].ID !=
		"beyond_stars" {

		t.Fatalf(
			"esperado beyond_stars, recebido %s",
			candidates[0].ID,
		)
	}
}

func TestRaidRotationIgnoresOtherRaritiesInHistory(
	t *testing.T,
) {

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	// Mythic, Sacred e Legendary pertencente a outra
	// fonte de progressão não devem bloquear o pool
	// Legendary ativo do Raid.
	recent :=
		[]string{
			"black_dragon",
			"ancient_titan",
			"bahamut",
		}

	candidates :=
		RaidRotationCandidates(
			catalog,
			RarityLegendary,
			recent,
		)

	if len(candidates) != 2 {
		t.Fatalf(
			"esperados 2 Legendary disponíveis, recebidos %d",
			len(candidates),
		)
	}

	found :=
		map[string]bool{}

	for _, boss := range candidates {
		found[boss.ID] = true
	}

	if !found["crimson_chimera"] ||
		!found["ash_cerberus"] {

		t.Fatalf(
			"pool Legendary inesperado: %+v",
			found,
		)
	}
}

func TestSelectRaidBoss(
	t *testing.T,
) {

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 50; i++ {

		boss, err :=
			SelectRaidBoss(
				catalog,
				nil,
			)

		if err != nil {
			t.Fatal(err)
		}

		if boss.ID == "" {
			t.Fatal(
				"sorteio retornou Boss vazio",
			)
		}

		if boss.Power <= 0 {
			t.Fatalf(
				"Boss %s possui PC inválido",
				boss.ID,
			)
		}
	}
}
