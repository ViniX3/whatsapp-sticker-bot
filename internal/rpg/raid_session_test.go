package rpg

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestCreateRaidSession(
	t *testing.T,
) {
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

	if session.ID <= 0 {
		t.Fatalf(
			"ID de Raid inválido: %d",
			session.ID,
		)
	}

	if session.Boss.ID !=
		"black_dragon" {

		t.Fatalf(
			"Boss inesperado: %s",
			session.Boss.ID,
		)
	}

	expectedStart :=
		now.Add(
			RaidJoinWindow,
		)

	if !session.StartsAt.Equal(
		expectedStart,
	) {
		t.Fatalf(
			"horário esperado %s, recebido %s",
			expectedStart,
			session.StartsAt,
		)
	}
}

func TestOnlyOneActiveRaidPerGroup(
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

	first :=
		mustRaidBoss(
			t,
			catalog,
			"black_dragon",
		)

	second :=
		mustRaidBoss(
			t,
			catalog,
			"leviathan",
		)

	if _, err :=
		CreateRaidSession(
			"group-a",
			first,
			now,
		); err != nil {

		t.Fatal(err)
	}

	_, err =
		CreateRaidSession(
			"group-a",
			second,
			now,
		)

	if !errors.Is(
		err,
		ErrRaidActiveExists,
	) {
		t.Fatalf(
			"esperado ErrRaidActiveExists, recebido %v",
			err,
		)
	}
}

func TestRaidSessionJoinPersistsPowerAndBonuses(
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

	boss :=
		mustRaidBoss(
			t,
			catalog,
			"black_dragon",
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

	participant :=
		RaidParticipant{
			JID: "player-a",

			Name: "Player A",

			CombatPower: 2000,

			Bonuses: RaidBonuses{
				BossDamagePercent: 10,

				BlessingChancePercent: 4,

				CrystalRewardPercent: 15,

				StellarStoneChancePercent: 3,

				DropChancePercent: 8,

				RareDropChancePercent: 5,

				DropRarityUpgradeChancePercent: 2,
			},
		}

	if err :=
		JoinActiveRaidSession(
			"group-a",
			participant,
			now,
		); err != nil {

		t.Fatal(err)
	}

	participants, err :=
		RaidSessionParticipants(
			session.ID,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(participants) != 1 {
		t.Fatalf(
			"esperado 1 participante, recebido %d",
			len(participants),
		)
	}

	stored :=
		participants[0]

	if stored.EffectivePower != 2200 {
		t.Fatalf(
			"PC efetivo esperado 2200, recebido %d",
			stored.EffectivePower,
		)
	}

	if stored.Bonuses.
		CrystalRewardPercent != 15 {

		t.Fatalf(
			"bônus de Cristal esperado 15, recebido %d",
			stored.Bonuses.
				CrystalRewardPercent,
		)
	}

	if stored.Bonuses.
		StellarStoneChancePercent != 3 {

		t.Fatalf(
			"bônus de Pedra Estelar esperado 3, recebido %d",
			stored.Bonuses.
				StellarStoneChancePercent,
		)
	}
}

func TestRaidSessionRejectsDuplicateParticipant(
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

	boss :=
		mustRaidBoss(
			t,
			catalog,
			"behemoth",
		)

	if _, err :=
		CreateRaidSession(
			"group-a",
			boss,
			now,
		); err != nil {

		t.Fatal(err)
	}

	participant :=
		RaidParticipant{
			JID: "player-a",

			CombatPower: 1000,
		}

	if err :=
		JoinActiveRaidSession(
			"group-a",
			participant,
			now,
		); err != nil {

		t.Fatal(err)
	}

	err =
		JoinActiveRaidSession(
			"group-a",
			participant,
			now,
		)

	if !errors.Is(
		err,
		ErrRaidAlreadyJoined,
	) {
		t.Fatalf(
			"esperado ErrRaidAlreadyJoined, recebido %v",
			err,
		)
	}
}

func TestRaidSessionCapacity(
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

	boss :=
		mustRaidBoss(
			t,
			catalog,
			"ancient_titan",
		)

	if _, err :=
		CreateRaidSession(
			"group-a",
			boss,
			now,
		); err != nil {

		t.Fatal(err)
	}

	for i := 0; i <
		RaidMaxParticipants; i++ {

		participant :=
			RaidParticipant{
				JID: fmtTestRaidJID(
					i,
				),

				CombatPower: 1000,
			}

		if err :=
			JoinActiveRaidSession(
				"group-a",
				participant,
				now,
			); err != nil {

			t.Fatal(err)
		}
	}

	err =
		JoinActiveRaidSession(
			"group-a",
			RaidParticipant{
				JID: "extra-player",

				CombatPower: 1000,
			},
			now,
		)

	if !errors.Is(
		err,
		ErrRaidLobbyFull,
	) {
		t.Fatalf(
			"esperado ErrRaidLobbyFull, recebido %v",
			err,
		)
	}
}

func TestRaidSessionRejectsLateJoin(
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

	boss :=
		mustRaidBoss(
			t,
			catalog,
			"black_dragon",
		)

	if _, err :=
		CreateRaidSession(
			"group-a",
			boss,
			now,
		); err != nil {

		t.Fatal(err)
	}

	err =
		JoinActiveRaidSession(
			"group-a",
			RaidParticipant{
				JID: "player-a",

				CombatPower: 1000,
			},
			now.Add(
				RaidJoinWindow,
			),
		)

	if !errors.Is(
		err,
		ErrRaidLobbyClosed,
	) {
		t.Fatalf(
			"esperado ErrRaidLobbyClosed, recebido %v",
			err,
		)
	}
}

func TestLoadActiveRaidLobbyRestoresCollectivePower(
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

	boss :=
		mustRaidBoss(
			t,
			catalog,
			"black_dragon",
		)

	if _, err :=
		CreateRaidSession(
			"group-a",
			boss,
			now,
		); err != nil {

		t.Fatal(err)
	}

	for i := 0; i < 4; i++ {
		if err :=
			JoinActiveRaidSession(
				"group-a",
				RaidParticipant{
					JID: fmtTestRaidJID(
						i,
					),

					CombatPower: 3000,
				},
				now,
			); err != nil {

			t.Fatal(err)
		}
	}

	lobby, err :=
		LoadActiveRaidLobby(
			"group-a",
		)

	if err != nil {
		t.Fatal(err)
	}

	if lobby.TotalPower() != 12000 {
		t.Fatalf(
			"PC coletivo esperado 12000, recebido %d",
			lobby.TotalPower(),
		)
	}

	if lobby.SuccessChance() != 50 {
		t.Fatalf(
			"chance esperada 50%%, recebida %d%%",
			lobby.SuccessChance(),
		)
	}
}

func TestCloseRaidSessionAllowsNewRaid(
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

	first :=
		mustRaidBoss(
			t,
			catalog,
			"black_dragon",
		)

	if _, err :=
		CreateRaidSession(
			"group-a",
			first,
			now,
		); err != nil {

		t.Fatal(err)
	}

	if err :=
		CloseActiveRaidSession(
			"group-a",
			RaidSessionResolved,
			now.Add(
				RaidJoinWindow,
			),
		); err != nil {

		t.Fatal(err)
	}

	second :=
		mustRaidBoss(
			t,
			catalog,
			"leviathan",
		)

	if _, err :=
		CreateRaidSession(
			"group-a",
			second,
			now.Add(
				RaidJoinWindow+
					time.Minute,
			),
		); err != nil {

		t.Fatalf(
			"segunda Raid deveria ser permitida: %v",
			err,
		)
	}
}

func fmtTestRaidJID(
	index int,
) string {
	return fmt.Sprintf(
		"player-%02d",
		index,
	)
}
