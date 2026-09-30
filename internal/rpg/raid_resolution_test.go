package rpg

import (
	"errors"
	"testing"
	"time"
)

func prepareRaidResolutionTest(
	t *testing.T,
	playerCount int,
	playerPower int,
) (
	int64,
	time.Time,
) {
	t.Helper()

	setupRaidHistoryTestDB(t)

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	boss :=
		mustRaidBoss(
			t,
			catalog,
			"black_dragon",
		)

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

	session, err :=
		CreateRaidSession(
			"group-a",
			boss,
			now,
		)

	if err != nil {
		t.Fatal(err)
	}

	for index := 0; index < playerCount; index++ {

		err :=
			JoinActiveRaidSession(
				"group-a",
				RaidParticipant{
					JID: fmtTestRaidJID(
						index,
					),

					Name: fmtTestRaidJID(
						index,
					),

					CombatPower: playerPower,
				},
				now.Add(
					time.Second,
				),
			)

		if err != nil {
			t.Fatal(err)
		}
	}

	return session.ID,
		session.StartsAt
}

func TestResolveRaidVictory(
	t *testing.T,
) {
	raidID,
		startsAt :=
		prepareRaidResolutionTest(
			t,
			4,
			3000,
		)

	result, err :=
		ResolveRaidSession(
			raidID,
			startsAt,
			50,
		)

	if err != nil {
		t.Fatal(err)
	}

	if result.ParticipantCount != 4 {
		t.Fatalf(
			"esperados 4 participantes, recebido %d",
			result.ParticipantCount,
		)
	}

	if result.TotalPower != 12000 {
		t.Fatalf(
			"PC coletivo esperado 12000, recebido %d",
			result.TotalPower,
		)
	}

	if result.SuccessChance != 50 {
		t.Fatalf(
			"chance esperada 50, recebida %d",
			result.SuccessChance,
		)
	}

	if !result.Won {
		t.Fatal(
			"roll 50 deveria vencer com 50% de chance",
		)
	}
}

func TestResolveRaidDefeat(
	t *testing.T,
) {
	raidID,
		startsAt :=
		prepareRaidResolutionTest(
			t,
			4,
			3000,
		)

	result, err :=
		ResolveRaidSession(
			raidID,
			startsAt,
			51,
		)

	if err != nil {
		t.Fatal(err)
	}

	if result.Won {
		t.Fatal(
			"roll 51 deveria perder com 50% de chance",
		)
	}
}

func TestRaidWithoutParticipantsAlwaysLoses(
	t *testing.T,
) {
	raidID,
		startsAt :=
		prepareRaidResolutionTest(
			t,
			0,
			0,
		)

	result, err :=
		ResolveRaidSession(
			raidID,
			startsAt,
			1,
		)

	if err != nil {
		t.Fatal(err)
	}

	if result.SuccessChance != 0 {
		t.Fatalf(
			"Raid vazia deveria ter 0%%, recebeu %d%%",
			result.SuccessChance,
		)
	}

	if result.Won {
		t.Fatal(
			"Raid vazia nunca deve vencer",
		)
	}
}

func TestRaidCannotResolveBeforeStart(
	t *testing.T,
) {
	raidID,
		startsAt :=
		prepareRaidResolutionTest(
			t,
			1,
			3000,
		)

	_, err :=
		ResolveRaidSession(
			raidID,
			startsAt.Add(
				-time.Second,
			),
			1,
		)

	if !errors.Is(
		err,
		ErrRaidNotDue,
	) {
		t.Fatalf(
			"esperado ErrRaidNotDue, recebido %v",
			err,
		)
	}
}

func TestRaidCannotResolveTwice(
	t *testing.T,
) {
	raidID,
		startsAt :=
		prepareRaidResolutionTest(
			t,
			1,
			3000,
		)

	if _, err :=
		ResolveRaidSession(
			raidID,
			startsAt,
			1,
		); err != nil {

		t.Fatal(err)
	}

	_, err :=
		ResolveRaidSession(
			raidID,
			startsAt.Add(
				time.Second,
			),
			1,
		)

	if !errors.Is(
		err,
		ErrRaidAlreadyResolved,
	) {
		t.Fatalf(
			"esperado ErrRaidAlreadyResolved, recebido %v",
			err,
		)
	}
}

func TestRaidResolutionAnnouncementPersistence(
	t *testing.T,
) {
	raidID,
		startsAt :=
		prepareRaidResolutionTest(
			t,
			1,
			3000,
		)

	if _, err :=
		ResolveRaidSession(
			raidID,
			startsAt,
			1,
		); err != nil {

		t.Fatal(err)
	}

	pending, err :=
		PendingRaidResolutionAnnouncements(
			10,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(pending) != 1 {
		t.Fatalf(
			"esperado 1 anúncio pendente, recebido %d",
			len(pending),
		)
	}

	if err :=
		MarkRaidResolutionAnnounced(
			raidID,
			startsAt,
		); err != nil {

		t.Fatal(err)
	}

	pending, err =
		PendingRaidResolutionAnnouncements(
			10,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(pending) != 0 {
		t.Fatalf(
			"esperados 0 anúncios pendentes, recebido %d",
			len(pending),
		)
	}
}
