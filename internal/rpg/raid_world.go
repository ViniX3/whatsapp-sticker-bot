package rpg

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

func RaidGroupHasPlayers(
	groupJID string,
) (
	bool,
	error,
) {
	groupJID =
		strings.TrimSpace(
			groupJID,
		)

	if groupJID == "" {
		return false,
			ErrInvalidRaidGroup
	}

	if err :=
		ensureSchema(); err != nil {

		return false, err
	}

	var count int

	err :=
		database.DB.QueryRow(
			`
			SELECT COUNT(*)
			FROM rpg_players
			WHERE group_jid = ?
			`,
			groupJID,
		).Scan(
			&count,
		)

	if err != nil {
		return false,
			fmt.Errorf(
				"erro verificando jogadores RPG do grupo: %w",
				err,
			)
	}

	return count > 0,
		nil
}

func RaidCanSpawnForGroup(
	groupJID string,
	now time.Time,
) (
	bool,
	error,
) {
	groupJID =
		strings.TrimSpace(
			groupJID,
		)

	if groupJID == "" {
		return false,
			ErrInvalidRaidGroup
	}

	if now.IsZero() {
		now = time.Now()
	}

	if err :=
		ensureRaidSessionSchema(); err != nil {

		return false, err
	}

	var activeCount int

	err :=
		database.DB.QueryRow(
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
		return false,
			fmt.Errorf(
				"erro verificando Boss ativo: %w",
				err,
			)
	}

	if activeCount > 0 {
		return false,
			nil
	}

	var resolvedAt time.Time

	err =
		database.DB.QueryRow(
			`
			SELECT resolved_at
			FROM rpg_raid_sessions
			WHERE group_jid = ?
			  AND status = 'RESOLVED'
			  AND resolved_at IS NOT NULL
			ORDER BY resolved_at DESC, id DESC
			LIMIT 1
			`,
			groupJID,
		).Scan(
			&resolvedAt,
		)

	if err == sql.ErrNoRows {
		return true,
			nil
	}

	if err != nil {
		return false,
			fmt.Errorf(
				"erro consultando último Raid Boss: %w",
				err,
			)
	}

	return !now.Before(
			resolvedAt.Add(
				RaidRespawnDelay,
			),
		),
		nil
}

func CreateDormantRaidSession(
	groupJID string,
	boss RaidBoss,
	now time.Time,
) (
	*RaidSession,
	error,
) {
	groupJID =
		strings.TrimSpace(
			groupJID,
		)

	if groupJID == "" {
		return nil,
			ErrInvalidRaidGroup
	}

	if err :=
		validateRaidBoss(
			boss,
		); err != nil {

		return nil, err
	}

	if now.IsZero() {
		now = time.Now()
	}

	if err :=
		ensureRaidSessionSchema(); err != nil {

		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando criação do Raid Boss: %w",
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
			RaidDormantWindow,
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
			string(
				boss.Rarity,
			),
			boss.Power,
			now,
			startsAt,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro criando Raid Boss permanente: %w",
				err,
			)
	}

	raidID, err :=
		result.LastInsertId()

	if err != nil {
		return nil, err
	}

	if err :=
		tx.Commit(); err != nil {

		return nil, err
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

func OpenDormantRaidForGroup(
	bossCatalog *RaidBossCatalog,
	itemCatalog *Catalog,
	groupJID string,
	now time.Time,
) (
	*OpenRaidResult,
	error,
) {
	if bossCatalog == nil {
		return nil,
			ErrInvalidRaidBossCatalog
	}

	boss, err :=
		SelectRaidBossForGroup(
			bossCatalog,
			groupJID,
		)

	if err != nil {
		return nil,
			err
	}

	scaledPower, err :=
		RaidScaledBossPower(
			groupJID,
			boss,
			itemCatalog,
		)

	if err != nil {
		return nil, err
	}

	boss.Power =
		scaledPower

	session, err :=
		CreateDormantRaidSession(
			groupJID,
			boss,
			now,
		)

	if err != nil {
		return nil, err
	}

	_, err =
		RecordRaidAnnouncement(
			groupJID,
			boss,
			now,
		)

	if err != nil {
		_ =
			CloseActiveRaidSession(
				groupJID,
				RaidSessionCancelled,
				now,
			)

		return nil,
			fmt.Errorf(
				"erro registrando Raid Boss automático: %w",
				err,
			)
	}

	return &OpenRaidResult{
			Session: session,

			Boss: boss,
		},
		nil
}
