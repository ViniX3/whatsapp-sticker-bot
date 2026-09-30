package rpg

import (
	"errors"
	"testing"
	"time"
)

func TestRaidBossCatalog(
	t *testing.T,
) {

	bosses :=
		RaidBosses()

	if len(bosses) != 34 {
		t.Fatalf(
			"esperados 34 Raid Bosses, encontrados %d",
			len(bosses),
		)
	}

	counts :=
		map[Rarity]int{}

	previousPower := 0

	for _, boss := range bosses {

		if boss.ID == "" {
			t.Fatal(
				"Raid Boss sem ID",
			)
		}

		if boss.Name == "" {
			t.Fatalf(
				"Raid Boss %s sem nome",
				boss.ID,
			)
		}

		if boss.Power <= 0 {
			t.Fatalf(
				"Raid Boss %s possui poder inválido",
				boss.ID,
			)
		}

		if boss.SetID == "" {
			t.Fatalf(
				"Raid Boss %s não possui SetID",
				boss.ID,
			)
		}

		if boss.Power <=
			previousPower {

			t.Fatalf(
				"progressão de poder inválida em %s: %d <= %d",
				boss.ID,
				boss.Power,
				previousPower,
			)
		}

		previousPower =
			boss.Power

		counts[boss.Rarity]++
	}

	if counts[RarityLegendary] != 16 {
		t.Fatalf(
			"esperados 16 Legendary, encontrados %d",
			counts[RarityLegendary],
		)
	}

	if counts[RarityMythic] != 15 {
		t.Fatalf(
			"esperados 15 Mythic, encontrados %d",
			counts[RarityMythic],
		)
	}

	if counts[RaritySacred] != 3 {
		t.Fatalf(
			"esperados 3 Sacred, encontrados %d",
			counts[RaritySacred],
		)
	}
}

func TestRaidBossByID(
	t *testing.T,
) {

	boss, exists :=
		RaidBossByID(
			"  ANCIENT_TITAN  ",
		)

	if !exists {
		t.Fatal(
			"Ancient Titan não encontrado",
		)
	}

	if boss.SetID !=
		"mythic_ancient_titan" {

		t.Fatalf(
			"SetID inesperado: %s",
			boss.SetID,
		)
	}
}

func TestNewRaidLobby(
	t *testing.T,
) {

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

	lobby, err :=
		NewRaidLobby(
			"group@test",
			"black_dragon",
			now,
		)

	if err != nil {
		t.Fatal(err)
	}

	if lobby.Boss.ID !=
		"black_dragon" {

		t.Fatalf(
			"boss inesperado: %s",
			lobby.Boss.ID,
		)
	}

	wantStartsAt :=
		now.Add(
			RaidJoinWindow,
		)

	if !lobby.StartsAt.Equal(
		wantStartsAt,
	) {

		t.Fatalf(
			"StartsAt inesperado: %s",
			lobby.StartsAt,
		)
	}
}

func TestRaidLobbyJoin(
	t *testing.T,
) {

	now :=
		time.Now()

	lobby, err :=
		NewRaidLobby(
			"group@test",
			"black_dragon",
			now,
		)

	if err != nil {
		t.Fatal(err)
	}

	err =
		lobby.Join(
			RaidParticipant{
				JID: "player@test",

				CombatPower: 2000,

				Bonuses: RaidBonuses{
					BossDamagePercent: 10,
				},
			},
			now,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(
		lobby.Participants,
	) != 1 {

		t.Fatalf(
			"esperado 1 participante, encontrado %d",
			len(lobby.Participants),
		)
	}

	participant :=
		lobby.Participants[0]

	if participant.EffectivePower !=
		2200 {

		t.Fatalf(
			"esperado PC efetivo 2200, recebido %d",
			participant.EffectivePower,
		)
	}

	if lobby.TotalPower() != 2200 {
		t.Fatalf(
			"poder total inesperado: %d",
			lobby.TotalPower(),
		)
	}
}

func TestRaidLobbyRejectsDuplicate(
	t *testing.T,
) {

	now :=
		time.Now()

	lobby, err :=
		NewRaidLobby(
			"group@test",
			"behemoth",
			now,
		)

	if err != nil {
		t.Fatal(err)
	}

	player :=
		RaidParticipant{
			JID: "player@test",

			CombatPower: 1000,
		}

	if err :=
		lobby.Join(
			player,
			now,
		); err != nil {

		t.Fatal(err)
	}

	err =
		lobby.Join(
			player,
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

func TestRaidLobbyRejectsClosedLobby(
	t *testing.T,
) {

	now :=
		time.Now()

	lobby, err :=
		NewRaidLobby(
			"group@test",
			"black_dragon",
			now,
		)

	if err != nil {
		t.Fatal(err)
	}

	err =
		lobby.Join(
			RaidParticipant{
				JID: "player@test",

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

func TestRaidLobbyCapacity(
	t *testing.T,
) {

	now :=
		time.Now()

	lobby, err :=
		NewRaidLobby(
			"group@test",
			"ancient_titan",
			now,
		)

	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < RaidMaxParticipants; i++ {

		err =
			lobby.Join(
				RaidParticipant{
					JID: string(
						rune(
							'A' + i,
						),
					),

					CombatPower: 1000,
				},
				now,
			)

		if err != nil {
			t.Fatal(err)
		}
	}

	err =
		lobby.Join(
			RaidParticipant{
				JID: "extra",

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

func TestRaidLobbySuccessChance(
	t *testing.T,
) {

	now :=
		time.Now()

	lobby, err :=
		NewRaidLobby(
			"group@test",
			"black_dragon",
			now,
		)

	if err != nil {
		t.Fatal(err)
	}

	// Boss = 12.000.
	// RAID = 12.000.
	// Chance esperada = 50%.
	for i := 0; i < 4; i++ {

		err =
			lobby.Join(
				RaidParticipant{
					JID: string(
						rune(
							'A' + i,
						),
					),

					CombatPower: 3000,
				},
				now,
			)

		if err != nil {
			t.Fatal(err)
		}
	}

	if got :=
		lobby.SuccessChance(); got != 50 {

		t.Fatalf(
			"esperado 50%%, recebido %d%%",
			got,
		)
	}
}
