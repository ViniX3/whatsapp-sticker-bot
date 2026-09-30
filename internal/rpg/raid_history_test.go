package rpg

import (
	"database/sql"
	"testing"
	"time"

	"whatsapp-sticker-bot/internal/database"

	_ "github.com/mattn/go-sqlite3"
)

func setupRaidHistoryTestDB(
	t *testing.T,
) {
	t.Helper()

	oldDB :=
		database.DB

	db, err :=
		sql.Open(
			"sqlite3",
			":memory:",
		)

	if err != nil {
		t.Fatal(err)
	}

	// SQLite :memory: é por conexão.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	database.DB =
		db

	t.Cleanup(
		func() {
			_ = db.Close()

			database.DB =
				oldDB
		},
	)
}

func mustRaidBoss(
	t *testing.T,
	catalog *RaidBossCatalog,
	bossID string,
) RaidBoss {
	t.Helper()

	boss, exists :=
		catalog.BossByID(
			bossID,
		)

	if !exists {
		t.Fatalf(
			"Boss %s não encontrado",
			bossID,
		)
	}

	return boss
}

func TestRaidHistoryStoresRecentBosses(
	t *testing.T,
) {
	setupRaidHistoryTestDB(t)

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	now :=
		time.Date(
			2026,
			9,
			30,
			12,
			0,
			0,
			0,
			time.UTC,
		)

	bossIDs :=
		[]string{
			"black_dragon",
			"storm_griffin",
			"frost_wyrm",
		}

	for index, bossID := range bossIDs {

		boss :=
			mustRaidBoss(
				t,
				catalog,
				bossID,
			)

		_, err :=
			RecordRaidAnnouncement(
				"group-a",
				boss,
				now.Add(
					time.Duration(index)*
						time.Minute,
				),
			)

		if err != nil {
			t.Fatal(err)
		}
	}

	recent, err :=
		RecentRaidBossIDs(
			"group-a",
			RarityLegendary,
			2,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(recent) != 2 {
		t.Fatalf(
			"esperados 2 Bosses, recebidos %d",
			len(recent),
		)
	}

	if recent[0] !=
		"storm_griffin" {

		t.Fatalf(
			"Boss antigo esperado storm_griffin, recebido %s",
			recent[0],
		)
	}

	if recent[1] !=
		"frost_wyrm" {

		t.Fatalf(
			"Boss recente esperado frost_wyrm, recebido %s",
			recent[1],
		)
	}
}

func TestRaidHistoryIsIsolatedByGroup(
	t *testing.T,
) {
	setupRaidHistoryTestDB(t)

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	now :=
		time.Now()

	blackDragon :=
		mustRaidBoss(
			t,
			catalog,
			"black_dragon",
		)

	leviathan :=
		mustRaidBoss(
			t,
			catalog,
			"leviathan",
		)

	if _, err :=
		RecordRaidAnnouncement(
			"group-a",
			blackDragon,
			now,
		); err != nil {

		t.Fatal(err)
	}

	if _, err :=
		RecordRaidAnnouncement(
			"group-b",
			leviathan,
			now,
		); err != nil {

		t.Fatal(err)
	}

	groupA, err :=
		RecentRaidBossIDs(
			"group-a",
			RarityLegendary,
			6,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(groupA) != 1 ||
		groupA[0] !=
			"black_dragon" {

		t.Fatalf(
			"histórico inesperado do grupo A: %v",
			groupA,
		)
	}

	groupB, err :=
		RecentRaidBossIDs(
			"group-b",
			RarityLegendary,
			6,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(groupB) != 1 ||
		groupB[0] !=
			"leviathan" {

		t.Fatalf(
			"histórico inesperado do grupo B: %v",
			groupB,
		)
	}
}

func TestRaidHistoryRotationLimits(
	t *testing.T,
) {
	setupRaidHistoryTestDB(t)

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	now :=
		time.Date(
			2026,
			9,
			30,
			12,
			0,
			0,
			0,
			time.UTC,
		)

	legendary :=
		[]string{
			"black_dragon",
			"storm_griffin",
			"frost_wyrm",
			"behemoth",
			"leviathan",
			"blood_moon_wolf",
			"ancient_demon",
		}

	mythic :=
		[]string{
			"phoenix",
			"ancient_titan",
			"world_serpent",
			"celestial_hydra",
			"abyss_king",
			"void_colossus",
		}

	sacred :=
		[]string{
			"bahamut",
			"azathor",
			"beyond_stars",
		}

	index := 0

	for _, bossID := range append(
		append(
			legendary,
			mythic...,
		),
		sacred...,
	) {

		boss :=
			mustRaidBoss(
				t,
				catalog,
				bossID,
			)

		if _, err :=
			RecordRaidAnnouncement(
				"group-a",
				boss,
				now.Add(
					time.Duration(index)*
						time.Minute,
				),
			); err != nil {

			t.Fatal(err)
		}

		index++
	}

	recent, err :=
		RecentRaidBossIDsForRotation(
			"group-a",
		)

	if err != nil {
		t.Fatal(err)
	}

	expected :=
		RaidLegendaryRecentBlock +
			RaidMythicRecentBlock +
			RaidSacredRecentBlock

	if len(recent) != expected {
		t.Fatalf(
			"esperados %d IDs recentes, recebidos %d",
			expected,
			len(recent),
		)
	}

	legendaryCandidates :=
		RaidRotationCandidates(
			catalog,
			RarityLegendary,
			recent,
		)

	if len(legendaryCandidates) != 2 {
		t.Fatalf(
			"esperados 2 Legendary ativos disponíveis, recebidos %d",
			len(legendaryCandidates),
		)
	}

	legendaryFound :=
		map[string]bool{}

	for _, boss := range legendaryCandidates {
		legendaryFound[boss.ID] = true
	}

	if !legendaryFound["crimson_chimera"] ||
		!legendaryFound["ash_cerberus"] {

		t.Fatalf(
			"pool Legendary ativo inesperado: %+v",
			legendaryFound,
		)
	}

	mythicCandidates :=
		RaidRotationCandidates(
			catalog,
			RarityMythic,
			recent,
		)

	if len(mythicCandidates) != 10 {
		t.Fatalf(
			"esperados 10 Mythic disponíveis, recebidos %d",
			len(mythicCandidates),
		)
	}

	sacredCandidates :=
		RaidRotationCandidates(
			catalog,
			RaritySacred,
			recent,
		)

	if len(sacredCandidates) != 1 {
		t.Fatalf(
			"esperado 1 Sacred disponível, recebidos %d",
			len(sacredCandidates),
		)
	}
}

func TestSelectRaidBossForGroup(
	t *testing.T,
) {
	setupRaidHistoryTestDB(t)

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 25; i++ {
		boss, err :=
			SelectRaidBossForGroup(
				catalog,
				"group-a",
			)

		if err != nil {
			t.Fatal(err)
		}

		if boss.ID == "" {
			t.Fatal(
				"sorteio retornou Boss vazio",
			)
		}
	}
}
