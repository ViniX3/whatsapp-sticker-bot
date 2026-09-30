package rpg

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

var (
	ErrRaidNotDue = errors.New(
		"Raid ainda não chegou ao horário de resolução",
	)

	ErrRaidAlreadyResolved = errors.New(
		"Raid já foi resolvida",
	)
)

type RaidResolutionResult struct {
	RaidID int64

	GroupJID string

	Boss RaidBoss

	ParticipantCount int

	TotalPower int

	SuccessChance int

	Roll int

	Won bool

	ResolvedAt time.Time

	AnnouncedAt *time.Time
}

func ensureRaidResolutionSchema() error {
	if err := ensureRaidSessionSchema(); err != nil {
		return err
	}

	statements :=
		[]string{
			`
			CREATE TABLE IF NOT EXISTS rpg_raid_results (
				raid_id INTEGER PRIMARY KEY,

				group_jid TEXT NOT NULL,

				boss_id TEXT NOT NULL,

				boss_power INTEGER NOT NULL
					CHECK (boss_power > 0),

				participant_count INTEGER NOT NULL
					CHECK (participant_count >= 0),

				total_power INTEGER NOT NULL
					CHECK (total_power >= 0),

				success_chance INTEGER NOT NULL
					CHECK (
						success_chance >= 0
						AND success_chance <= 99
					),

				roll INTEGER NOT NULL
					CHECK (
						roll >= 1
						AND roll <= 100
					),

				won INTEGER NOT NULL
					CHECK (
						won IN (0, 1)
					),

				resolved_at DATETIME NOT NULL,

				announced_at DATETIME,

				created_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,

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
				idx_rpg_raid_results_pending
			ON rpg_raid_results (
				announced_at,
				resolved_at
			);
			`,
		}

	for _, statement := range statements {

		if _, err :=
			database.DB.Exec(
				statement,
			); err != nil {

			return fmt.Errorf(
				"erro criando schema de resolução Raid: %w",
				err,
			)
		}
	}

	return nil
}

func RollRaidResolution() (
	int,
	error,
) {
	value, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(100),
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro sorteando resultado da Raid: %w",
				err,
			)
	}

	return int(
			value.Int64(),
		) + 1,
		nil
}

func DueRaidSessions(
	now time.Time,
	limit int,
) (
	[]RaidSession,
	error,
) {
	if now.IsZero() {
		now = time.Now()
	}

	if limit <= 0 {
		limit = 20
	}

	if err :=
		ensureRaidSessionSchema(); err != nil {

		return nil,
			err
	}

	rows, err :=
		database.DB.Query(
			`
			SELECT
				id,
				group_jid,
				boss_id,
				rarity,
				boss_power,
				status,
				created_at,
				starts_at
			FROM rpg_raid_sessions
			WHERE status = 'OPEN'
			  AND starts_at <= ?
			ORDER BY starts_at ASC, id ASC
			LIMIT ?
			`,
			now,
			limit,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando Raids vencidas: %w",
				err,
			)
	}

	defer rows.Close()

	sessions :=
		make(
			[]RaidSession,
			0,
		)

	for rows.Next() {
		var (
			session RaidSession

			bossID string

			rarity string

			bossPower int
		)

		if err :=
			rows.Scan(
				&session.ID,
				&session.GroupJID,
				&bossID,
				&rarity,
				&bossPower,
				&session.Status,
				&session.CreatedAt,
				&session.StartsAt,
			); err != nil {

			return nil,
				fmt.Errorf(
					"erro lendo Raid vencida: %w",
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

		boss.Rarity =
			Rarity(
				rarity,
			)

		boss.Power =
			bossPower

		session.Boss =
			boss

		sessions =
			append(
				sessions,
				session,
			)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"erro percorrendo Raids vencidas: %w",
				err,
			)
	}

	return sessions,
		nil
}

func ResolveRaidSession(
	raidID int64,
	now time.Time,
	roll int,
) (
	*RaidResolutionResult,
	error,
) {
	if raidID <= 0 {
		return nil,
			ErrRaidSessionNotFound
	}

	if roll < 1 ||
		roll > 100 {

		return nil,
			fmt.Errorf(
				"roll de Raid inválido: %d",
				roll,
			)
	}

	if now.IsZero() {
		now = time.Now()
	}

	if err :=
		ensureRaidResolutionSchema(); err != nil {

		return nil,
			err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando resolução da Raid: %w",
				err,
			)
	}

	defer tx.Rollback()

	var (
		groupJID string

		bossID string

		rarity string

		bossPower int

		startsAt time.Time
	)

	err =
		tx.QueryRow(
			`
			SELECT
				group_jid,
				boss_id,
				rarity,
				boss_power,
				starts_at
			FROM rpg_raid_sessions
			WHERE id = ?
			  AND status = 'OPEN'
			`,
			raidID,
		).Scan(
			&groupJID,
			&bossID,
			&rarity,
			&bossPower,
			&startsAt,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		var resultCount int

		checkErr :=
			tx.QueryRow(
				`
				SELECT COUNT(*)
				FROM rpg_raid_results
				WHERE raid_id = ?
				`,
				raidID,
			).Scan(
				&resultCount,
			)

		if checkErr == nil &&
			resultCount > 0 {

			return nil,
				ErrRaidAlreadyResolved
		}

		return nil,
			ErrRaidSessionNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando Raid para resolução: %w",
				err,
			)
	}

	if now.Before(
		startsAt,
	) {
		return nil,
			ErrRaidNotDue
	}

	var (
		participantCount int

		totalPower int
	)

	err =
		tx.QueryRow(
			`
			SELECT
				COUNT(*),
				COALESCE(
					SUM(effective_power),
					0
				)
			FROM rpg_raid_participants
			WHERE raid_id = ?
			`,
			raidID,
		).Scan(
			&participantCount,
			&totalPower,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro calculando poder coletivo da Raid: %w",
				err,
			)
	}

	successChance := 0

	if participantCount > 0 {
		successChance =
			RaidSuccessChance(
				totalPower,
				bossPower,
			)
	}

	won :=
		participantCount > 0 &&
			roll <= successChance

	updateResult, err :=
		tx.Exec(
			`
			UPDATE rpg_raid_sessions
			SET
				status = 'RESOLVED',
				resolved_at = ?
			WHERE id = ?
			  AND status = 'OPEN'
			`,
			now,
			raidID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro encerrando sessão da Raid: %w",
				err,
			)
	}

	affected, err :=
		updateResult.RowsAffected()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro verificando resolução da Raid: %w",
				err,
			)
	}

	if affected != 1 {
		return nil,
			ErrRaidAlreadyResolved
	}

	_, err =
		tx.Exec(
			`
			INSERT INTO rpg_raid_results (
				raid_id,
				group_jid,
				boss_id,
				boss_power,
				participant_count,
				total_power,
				success_chance,
				roll,
				won,
				resolved_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
			raidID,
			groupJID,
			bossID,
			bossPower,
			participantCount,
			totalPower,
			successChance,
			roll,
			won,
			now,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro persistindo resultado da Raid: %w",
				err,
			)
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando resultado da Raid: %w",
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

	boss.Rarity =
		Rarity(
			rarity,
		)

	boss.Power =
		bossPower

	return &RaidResolutionResult{
			RaidID: raidID,

			GroupJID: groupJID,

			Boss: boss,

			ParticipantCount: participantCount,

			TotalPower: totalPower,

			SuccessChance: successChance,

			Roll: roll,

			Won: won,

			ResolvedAt: now,
		},
		nil
}

func PendingRaidResolutionAnnouncements(
	limit int,
) (
	[]RaidResolutionResult,
	error,
) {
	if limit <= 0 {
		limit = 20
	}

	if err :=
		ensureRaidResolutionSchema(); err != nil {

		return nil,
			err
	}

	rows, err :=
		database.DB.Query(
			`
			SELECT
				raid_id,
				group_jid,
				boss_id,
				boss_power,
				participant_count,
				total_power,
				success_chance,
				roll,
				won,
				resolved_at
			FROM rpg_raid_results
			WHERE announced_at IS NULL
			ORDER BY resolved_at ASC, raid_id ASC
			LIMIT ?
			`,
			limit,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando resultados pendentes de Raid: %w",
				err,
			)
	}

	defer rows.Close()

	results :=
		make(
			[]RaidResolutionResult,
			0,
		)

	for rows.Next() {
		var (
			result RaidResolutionResult

			bossID string

			bossPower int
		)

		if err :=
			rows.Scan(
				&result.RaidID,
				&result.GroupJID,
				&bossID,
				&bossPower,
				&result.ParticipantCount,
				&result.TotalPower,
				&result.SuccessChance,
				&result.Roll,
				&result.Won,
				&result.ResolvedAt,
			); err != nil {

			return nil,
				fmt.Errorf(
					"erro lendo resultado pendente da Raid: %w",
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

		boss.Power =
			bossPower

		result.Boss =
			boss

		results =
			append(
				results,
				result,
			)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"erro percorrendo resultados pendentes: %w",
				err,
			)
	}

	return results,
		nil
}

func MarkRaidResolutionAnnounced(
	raidID int64,
	now time.Time,
) error {
	if raidID <= 0 {
		return ErrRaidSessionNotFound
	}

	if now.IsZero() {
		now = time.Now()
	}

	if err :=
		ensureRaidResolutionSchema(); err != nil {

		return err
	}

	result, err :=
		database.DB.Exec(
			`
			UPDATE rpg_raid_results
			SET announced_at = ?
			WHERE raid_id = ?
			  AND announced_at IS NULL
			`,
			now,
			raidID,
		)

	if err != nil {
		return fmt.Errorf(
			"erro marcando anúncio da Raid: %w",
			err,
		)
	}

	affected, err :=
		result.RowsAffected()

	if err != nil {
		return fmt.Errorf(
			"erro verificando anúncio da Raid: %w",
			err,
		)
	}

	if affected == 0 {
		return nil
	}

	return nil
}
