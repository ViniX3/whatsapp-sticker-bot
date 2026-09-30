package rpg

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

type RaidSessionStatus string

const (
	RaidSessionOpen RaidSessionStatus = "OPEN"

	RaidSessionResolved RaidSessionStatus = "RESOLVED"

	RaidSessionCancelled RaidSessionStatus = "CANCELLED"
)

var (
	ErrRaidActiveExists = errors.New(
		"já existe uma Raid ativa neste grupo",
	)

	ErrRaidSessionNotFound = errors.New(
		"Raid ativa não encontrada",
	)

	ErrInvalidRaidSessionStatus = errors.New(
		"status de Raid inválido",
	)
)

type RaidSession struct {
	ID int64

	GroupJID string

	Boss RaidBoss

	Status RaidSessionStatus

	CreatedAt time.Time

	StartsAt time.Time

	ResolvedAt *time.Time
}

func ensureRaidSessionSchema() error {
	if err := ensureSchema(); err != nil {
		return err
	}

	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS rpg_raid_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,

			group_jid TEXT NOT NULL,

			boss_id TEXT NOT NULL,

			rarity TEXT NOT NULL
				CHECK (
					rarity IN (
						'LEGENDARY',
						'MYTHIC',
						'SACRED'
					)
				),

			boss_power INTEGER NOT NULL
				CHECK (boss_power > 0),

			status TEXT NOT NULL
				DEFAULT 'OPEN'
				CHECK (
					status IN (
						'OPEN',
						'RESOLVED',
						'CANCELLED'
					)
				),

			created_at DATETIME NOT NULL,

			starts_at DATETIME NOT NULL,

			resolved_at DATETIME
		);
		`,

		`
		CREATE UNIQUE INDEX IF NOT EXISTS
			idx_rpg_raid_sessions_one_open_per_group
		ON rpg_raid_sessions (
			group_jid
		)
		WHERE status = 'OPEN';
		`,

		`
		CREATE INDEX IF NOT EXISTS
			idx_rpg_raid_sessions_group_status
		ON rpg_raid_sessions (
			group_jid,
			status,
			created_at
		);
		`,

		`
		CREATE TABLE IF NOT EXISTS rpg_raid_participants (
			raid_id INTEGER NOT NULL,

			jid TEXT NOT NULL,

			name TEXT NOT NULL
				DEFAULT '',

			combat_power INTEGER NOT NULL
				CHECK (combat_power > 0),

			effective_power INTEGER NOT NULL
				CHECK (effective_power > 0),

			boss_damage_percent INTEGER NOT NULL
				DEFAULT 0
				CHECK (boss_damage_percent >= 0),

			blessing_chance_percent INTEGER NOT NULL
				DEFAULT 0
				CHECK (blessing_chance_percent >= 0),

			crystal_reward_percent INTEGER NOT NULL
				DEFAULT 0
				CHECK (crystal_reward_percent >= 0),

			stellar_stone_chance_percent INTEGER NOT NULL
				DEFAULT 0
				CHECK (stellar_stone_chance_percent >= 0),

			drop_chance_percent INTEGER NOT NULL
				DEFAULT 0
				CHECK (drop_chance_percent >= 0),

			rare_drop_chance_percent INTEGER NOT NULL
				DEFAULT 0
				CHECK (rare_drop_chance_percent >= 0),

			drop_rarity_upgrade_chance_percent INTEGER NOT NULL
				DEFAULT 0
				CHECK (
					drop_rarity_upgrade_chance_percent >= 0
				),

			joined_at DATETIME NOT NULL,

			PRIMARY KEY (
				raid_id,
				jid
			),

			FOREIGN KEY (
				raid_id
			)
				REFERENCES rpg_raid_sessions (
					id
				)
				ON DELETE CASCADE
		);
		`,

		`
		CREATE INDEX IF NOT EXISTS
			idx_rpg_raid_participants_joined
		ON rpg_raid_participants (
			raid_id,
			joined_at
		);
		`,
	}

	for _, statement := range statements {
		if _, err := database.DB.Exec(statement); err != nil {
			return fmt.Errorf(
				"erro criando schema de sessão Raid: %w",
				err,
			)
		}
	}

	return nil
}

func CreateRaidSession(
	groupJID string,
	boss RaidBoss,
	now time.Time,
) (
	*RaidSession,
	error,
) {
	groupJID = strings.TrimSpace(groupJID)

	if groupJID == "" {
		return nil,
			ErrInvalidRaidGroup
	}

	if err := validateRaidBoss(boss); err != nil {
		return nil,
			err
	}

	if now.IsZero() {
		now = time.Now()
	}

	if err := ensureRaidSessionSchema(); err != nil {
		return nil,
			err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando criação da Raid: %w",
				err,
			)
	}

	defer tx.Rollback()

	var activeCount int

	err =
		tx.QueryRow(
			`
			SELECT COUNT(*)
			FROM rpg_raid_sessions
			WHERE group_jid = ?
			  AND status = 'OPEN'
			`,
			groupJID,
		).Scan(
			&activeCount,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro verificando Raid ativa: %w",
				err,
			)
	}

	if activeCount > 0 {
		return nil,
			ErrRaidActiveExists
	}

	startsAt :=
		now.Add(
			RaidJoinWindow,
		)

	result, err :=
		tx.Exec(
			`
			INSERT INTO rpg_raid_sessions (
				group_jid,
				boss_id,
				rarity,
				boss_power,
				status,
				created_at,
				starts_at
			)
			VALUES (?, ?, ?, ?, 'OPEN', ?, ?)
			`,
			groupJID,
			boss.ID,
			string(boss.Rarity),
			boss.Power,
			now,
			startsAt,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro criando sessão Raid: %w",
				err,
			)
	}

	raidID, err :=
		result.LastInsertId()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro obtendo ID da Raid: %w",
				err,
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando criação da Raid: %w",
				err,
			)
	}

	return &RaidSession{
			ID: raidID,

			GroupJID: groupJID,

			Boss: boss,

			Status: RaidSessionOpen,

			CreatedAt: now,

			StartsAt: startsAt,
		},
		nil
}

func GetActiveRaidSession(
	groupJID string,
) (
	*RaidSession,
	error,
) {
	groupJID = strings.TrimSpace(groupJID)

	if groupJID == "" {
		return nil,
			ErrInvalidRaidGroup
	}

	if err := ensureRaidSessionSchema(); err != nil {
		return nil,
			err
	}

	var (
		session RaidSession

		bossID string

		rarity string

		bossPower int

		resolvedAt sql.NullTime
	)

	err :=
		database.DB.QueryRow(
			`
			SELECT
				id,
				boss_id,
				rarity,
				boss_power,
				status,
				created_at,
				starts_at,
				resolved_at
			FROM rpg_raid_sessions
			WHERE group_jid = ?
			  AND status = 'OPEN'
			ORDER BY id DESC
			LIMIT 1
			`,
			groupJID,
		).Scan(
			&session.ID,
			&bossID,
			&rarity,
			&bossPower,
			&session.Status,
			&session.CreatedAt,
			&session.StartsAt,
			&resolvedAt,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			ErrRaidSessionNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando Raid ativa: %w",
				err,
			)
	}

	boss, exists :=
		RaidBossByID(
			bossID,
		)

	if !exists {
		return nil,
			fmt.Errorf(
				"%w: %s",
				ErrRaidBossNotFound,
				bossID,
			)
	}

	// Poder e raridade são congelados no início da Raid.
	boss.Rarity =
		Rarity(
			rarity,
		)

	boss.Power =
		bossPower

	session.GroupJID =
		groupJID

	session.Boss =
		boss

	if resolvedAt.Valid {
		value :=
			resolvedAt.Time

		session.ResolvedAt =
			&value
	}

	return &session,
		nil
}

func JoinActiveRaidSession(
	groupJID string,
	participant RaidParticipant,
	now time.Time,
) error {
	groupJID = strings.TrimSpace(groupJID)

	participant.JID =
		strings.TrimSpace(
			participant.JID,
		)

	participant.Name =
		strings.TrimSpace(
			participant.Name,
		)

	if groupJID == "" {
		return ErrInvalidRaidGroup
	}

	if participant.JID == "" ||
		participant.CombatPower <= 0 {

		return ErrRaidInvalidParticipant
	}

	if now.IsZero() {
		now = time.Now()
	}

	if err := ensureRaidSessionSchema(); err != nil {
		return err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return fmt.Errorf(
			"erro iniciando entrada na Raid: %w",
			err,
		)
	}

	defer tx.Rollback()

	var (
		raidID int64

		startsAt time.Time
	)

	err =
		tx.QueryRow(
			`
			SELECT
				id,
				starts_at
			FROM rpg_raid_sessions
			WHERE group_jid = ?
			  AND status = 'OPEN'
			ORDER BY id DESC
			LIMIT 1
			`,
			groupJID,
		).Scan(
			&raidID,
			&startsAt,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return ErrRaidSessionNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"erro consultando Raid para entrada: %w",
			err,
		)
	}

	if !now.Before(
		startsAt,
	) {
		return ErrRaidLobbyClosed
	}

	var duplicateCount int

	err =
		tx.QueryRow(
			`
			SELECT COUNT(*)
			FROM rpg_raid_participants
			WHERE raid_id = ?
			  AND jid = ?
			`,
			raidID,
			participant.JID,
		).Scan(
			&duplicateCount,
		)

	if err != nil {
		return fmt.Errorf(
			"erro verificando participante da Raid: %w",
			err,
		)
	}

	if duplicateCount > 0 {
		return ErrRaidAlreadyJoined
	}

	var participantCount int

	err =
		tx.QueryRow(
			`
			SELECT COUNT(*)
			FROM rpg_raid_participants
			WHERE raid_id = ?
			`,
			raidID,
		).Scan(
			&participantCount,
		)

	if err != nil {
		return fmt.Errorf(
			"erro contando participantes da Raid: %w",
			err,
		)
	}

	// Raid V2:
	// uma sessão adormecida só inicia o relógio quando
	// o primeiro aventureiro realmente entrar.
	if participantCount == 0 &&
		startsAt.After(
			now.Add(
				24*time.Hour,
			),
		) {

		startsAt =
			now.Add(
				RaidJoinWindow,
			)

		_, err =
			tx.Exec(
				`
				UPDATE rpg_raid_sessions
				SET starts_at = ?
				WHERE id = ?
				  AND status = 'OPEN'
				`,
				startsAt,
				raidID,
			)

		if err != nil {
			return fmt.Errorf(
				"erro iniciando contador da Raid: %w",
				err,
			)
		}
	}

	if participantCount >=
		RaidMaxParticipants {

		return ErrRaidLobbyFull
	}

	participant.EffectivePower =
		RaidEffectiveBossPower(
			participant.CombatPower,
			participant.Bonuses.
				BossDamagePercent,
		)

	participant.JoinedAt =
		now

	_, err =
		tx.Exec(
			`
			INSERT INTO rpg_raid_participants (
				raid_id,
				jid,
				name,
				combat_power,
				effective_power,
				boss_damage_percent,
				blessing_chance_percent,
				crystal_reward_percent,
				stellar_stone_chance_percent,
				drop_chance_percent,
				rare_drop_chance_percent,
				drop_rarity_upgrade_chance_percent,
				joined_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
			raidID,
			participant.JID,
			participant.Name,
			participant.CombatPower,
			participant.EffectivePower,
			participant.Bonuses.
				BossDamagePercent,
			participant.Bonuses.
				BlessingChancePercent,
			participant.Bonuses.
				CrystalRewardPercent,
			participant.Bonuses.
				StellarStoneChancePercent,
			participant.Bonuses.
				DropChancePercent,
			participant.Bonuses.
				RareDropChancePercent,
			participant.Bonuses.
				DropRarityUpgradeChancePercent,
			participant.JoinedAt,
		)

	if err != nil {
		return fmt.Errorf(
			"erro adicionando participante à Raid: %w",
			err,
		)
	}

	if err :=
		tx.Commit(); err != nil {

		return fmt.Errorf(
			"erro confirmando entrada na Raid: %w",
			err,
		)
	}

	return nil
}

func RaidSessionParticipants(
	raidID int64,
) (
	[]RaidParticipant,
	error,
) {
	if raidID <= 0 {
		return nil,
			ErrRaidSessionNotFound
	}

	if err := ensureRaidSessionSchema(); err != nil {
		return nil,
			err
	}

	rows, err :=
		database.DB.Query(
			`
			SELECT
				jid,
				name,
				combat_power,
				effective_power,
				boss_damage_percent,
				blessing_chance_percent,
				crystal_reward_percent,
				stellar_stone_chance_percent,
				drop_chance_percent,
				rare_drop_chance_percent,
				drop_rarity_upgrade_chance_percent,
				joined_at
			FROM rpg_raid_participants
			WHERE raid_id = ?
			ORDER BY joined_at ASC, jid ASC
			`,
			raidID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando participantes da Raid: %w",
				err,
			)
	}

	defer rows.Close()

	participants :=
		make(
			[]RaidParticipant,
			0,
			RaidMaxParticipants,
		)

	for rows.Next() {
		var participant RaidParticipant

		err :=
			rows.Scan(
				&participant.JID,
				&participant.Name,
				&participant.CombatPower,
				&participant.EffectivePower,
				&participant.Bonuses.
					BossDamagePercent,
				&participant.Bonuses.
					BlessingChancePercent,
				&participant.Bonuses.
					CrystalRewardPercent,
				&participant.Bonuses.
					StellarStoneChancePercent,
				&participant.Bonuses.
					DropChancePercent,
				&participant.Bonuses.
					RareDropChancePercent,
				&participant.Bonuses.
					DropRarityUpgradeChancePercent,
				&participant.JoinedAt,
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro lendo participante da Raid: %w",
					err,
				)
		}

		participants =
			append(
				participants,
				participant,
			)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"erro percorrendo participantes da Raid: %w",
				err,
			)
	}

	return participants,
		nil
}

func LoadActiveRaidLobby(
	groupJID string,
) (
	*RaidLobby,
	error,
) {
	session, err :=
		GetActiveRaidSession(
			groupJID,
		)

	if err != nil {
		return nil,
			err
	}

	participants, err :=
		RaidSessionParticipants(
			session.ID,
		)

	if err != nil {
		return nil,
			err
	}

	return &RaidLobby{
			GroupJID: session.GroupJID,

			Boss: session.Boss,

			CreatedAt: session.CreatedAt,

			StartsAt: session.StartsAt,

			Participants: participants,
		},
		nil
}

func CloseActiveRaidSession(
	groupJID string,
	status RaidSessionStatus,
	now time.Time,
) error {
	groupJID = strings.TrimSpace(groupJID)

	if groupJID == "" {
		return ErrInvalidRaidGroup
	}

	switch status {

	case RaidSessionResolved,
		RaidSessionCancelled:

	default:

		return ErrInvalidRaidSessionStatus
	}

	if now.IsZero() {
		now = time.Now()
	}

	if err := ensureRaidSessionSchema(); err != nil {
		return err
	}

	result, err :=
		database.DB.Exec(
			`
			UPDATE rpg_raid_sessions
			SET
				status = ?,
				resolved_at = ?
			WHERE group_jid = ?
			  AND status = 'OPEN'
			`,
			string(status),
			now,
			groupJID,
		)

	if err != nil {
		return fmt.Errorf(
			"erro encerrando Raid: %w",
			err,
		)
	}

	affected, err :=
		result.RowsAffected()

	if err != nil {
		return fmt.Errorf(
			"erro verificando encerramento da Raid: %w",
			err,
		)
	}

	if affected == 0 {
		return ErrRaidSessionNotFound
	}

	return nil
}
