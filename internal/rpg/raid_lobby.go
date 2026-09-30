package rpg

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrRaidLobbyClosed = errors.New(
		"lobby da raid está fechado",
	)

	ErrRaidLobbyFull = errors.New(
		"lobby da raid está lotado",
	)

	ErrRaidAlreadyJoined = errors.New(
		"jogador já entrou na raid",
	)

	ErrRaidInvalidParticipant = errors.New(
		"participante da raid inválido",
	)

	ErrRaidInvalidLobby = errors.New(
		"lobby da raid inválido",
	)
)

type RaidParticipant struct {
	JID string

	Name string

	CombatPower int

	EffectivePower int

	Bonuses RaidBonuses

	JoinedAt time.Time
}

type RaidLobby struct {
	GroupJID string

	Boss RaidBoss

	CreatedAt time.Time

	StartsAt time.Time

	Participants []RaidParticipant
}

func NewRaidLobby(
	groupJID string,
	bossID string,
	now time.Time,
) (
	*RaidLobby,
	error,
) {

	groupJID =
		strings.TrimSpace(
			groupJID,
		)

	if groupJID == "" {
		return nil,
			ErrRaidInvalidLobby
	}

	boss, exists :=
		RaidBossByID(
			bossID,
		)

	if !exists {
		return nil,
			ErrRaidBossNotFound
	}

	return &RaidLobby{
			GroupJID: groupJID,

			Boss: boss,

			CreatedAt: now,

			StartsAt: now.Add(
				RaidJoinWindow,
			),

			Participants: make(
				[]RaidParticipant,
				0,
				RaidMaxParticipants,
			),
		},
		nil
}

func (
	lobby *RaidLobby,
) IsOpen(
	now time.Time,
) bool {

	if lobby == nil {
		return false
	}

	return now.Before(
		lobby.StartsAt,
	)
}

func (
	lobby *RaidLobby,
) HasParticipant(
	jid string,
) bool {

	if lobby == nil {
		return false
	}

	jid =
		strings.TrimSpace(
			jid,
		)

	for _, participant := range lobby.Participants {

		if participant.JID ==
			jid {

			return true
		}
	}

	return false
}

func (
	lobby *RaidLobby,
) Join(
	participant RaidParticipant,
	now time.Time,
) error {

	if lobby == nil {
		return ErrRaidInvalidLobby
	}

	participant.JID =
		strings.TrimSpace(
			participant.JID,
		)

	if participant.JID == "" ||
		participant.CombatPower <= 0 {

		return ErrRaidInvalidParticipant
	}

	if !lobby.IsOpen(now) {
		return ErrRaidLobbyClosed
	}

	if lobby.HasParticipant(
		participant.JID,
	) {

		return ErrRaidAlreadyJoined
	}

	if len(lobby.Participants) >=
		RaidMaxParticipants {

		return ErrRaidLobbyFull
	}

	participant.EffectivePower =
		RaidEffectiveBossPower(
			participant.CombatPower,
			participant.Bonuses.
				BossDamagePercent,
		)

	participant.JoinedAt = now

	lobby.Participants =
		append(
			lobby.Participants,
			participant,
		)

	return nil
}

func (
	lobby *RaidLobby,
) TotalPower() int {

	if lobby == nil {
		return 0
	}

	total :=
		int64(0)

	for _, participant := range lobby.Participants {

		total +=
			int64(
				participant.
					EffectivePower,
			)
	}

	if total <= 0 {
		return 0
	}

	return int(total)
}

func (
	lobby *RaidLobby,
) SuccessChance() int {

	if lobby == nil {
		return 0
	}

	return RaidSuccessChance(
		lobby.TotalPower(),
		lobby.Boss.Power,
	)
}
