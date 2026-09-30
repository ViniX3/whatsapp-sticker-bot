package rpg

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

var (
	ErrInvalidRaidGroup = errors.New(
		"grupo da Raid inválido",
	)

	ErrInvalidRaidHistoryLimit = errors.New(
		"limite de histórico da Raid inválido",
	)
)

type RaidHistoryEntry struct {
	ID int64

	GroupJID string

	BossID string

	Rarity Rarity

	BossPower int

	AnnouncedAt time.Time
}

func ensureRaidHistorySchema() error {
	if err := ensureSchema(); err != nil {
		return err
	}

	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS rpg_raid_history (
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

			announced_at DATETIME NOT NULL,

			created_at DATETIME NOT NULL
				DEFAULT CURRENT_TIMESTAMP
		);
		`,

		`
		CREATE INDEX IF NOT EXISTS
			idx_rpg_raid_history_group_time
		ON rpg_raid_history (
			group_jid,
			announced_at,
			id
		);
		`,

		`
		CREATE INDEX IF NOT EXISTS
			idx_rpg_raid_history_group_rarity
		ON rpg_raid_history (
			group_jid,
			rarity,
			announced_at,
			id
		);
		`,
	}

	for _, statement := range statements {
		if _, err := database.DB.Exec(statement); err != nil {
			return fmt.Errorf(
				"erro criando schema de histórico de Raids: %w",
				err,
			)
		}
	}

	return nil
}

func RecordRaidAnnouncement(
	groupJID string,
	boss RaidBoss,
	announcedAt time.Time,
) (
	int64,
	error,
) {
	groupJID = strings.TrimSpace(groupJID)

	if groupJID == "" {
		return 0,
			ErrInvalidRaidGroup
	}

	if err := validateRaidBoss(boss); err != nil {
		return 0,
			err
	}

	if announcedAt.IsZero() {
		announcedAt = time.Now()
	}

	if err := ensureRaidHistorySchema(); err != nil {
		return 0,
			err
	}

	result, err :=
		database.DB.Exec(
			`
			INSERT INTO rpg_raid_history (
				group_jid,
				boss_id,
				rarity,
				boss_power,
				announced_at
			)
			VALUES (?, ?, ?, ?, ?)
			`,
			groupJID,
			boss.ID,
			string(boss.Rarity),
			boss.Power,
			announcedAt,
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro registrando anúncio da Raid: %w",
				err,
			)
	}

	id, err :=
		result.LastInsertId()

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro obtendo ID do histórico da Raid: %w",
				err,
			)
	}

	return id,
		nil
}

func RecentRaidBossIDs(
	groupJID string,
	rarity Rarity,
	limit int,
) (
	[]string,
	error,
) {
	groupJID = strings.TrimSpace(groupJID)

	if groupJID == "" {
		return nil,
			ErrInvalidRaidGroup
	}

	switch rarity {
	case RarityLegendary,
		RarityMythic,
		RaritySacred:

	default:
		return nil,
			fmt.Errorf(
				"raridade de Raid inválida: %s",
				rarity,
			)
	}

	if limit <= 0 {
		return nil,
			ErrInvalidRaidHistoryLimit
	}

	if err := ensureRaidHistorySchema(); err != nil {
		return nil,
			err
	}

	rows, err :=
		database.DB.Query(
			`
			SELECT boss_id
			FROM rpg_raid_history
			WHERE group_jid = ?
			  AND rarity = ?
			ORDER BY announced_at DESC, id DESC
			LIMIT ?
			`,
			groupJID,
			string(rarity),
			limit,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando histórico de Raids: %w",
				err,
			)
	}

	defer rows.Close()

	// A consulta retorna do mais novo para o mais antigo.
	// RaidRotationCandidates espera histórico cronológico,
	// portanto invertemos ao final.
	recent :=
		make(
			[]string,
			0,
			limit,
		)

	for rows.Next() {
		var bossID string

		if err :=
			rows.Scan(
				&bossID,
			); err != nil {

			return nil,
				fmt.Errorf(
					"erro lendo histórico de Raids: %w",
					err,
				)
		}

		recent =
			append(
				recent,
				bossID,
			)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"erro percorrendo histórico de Raids: %w",
				err,
			)
	}

	for left, right :=
		0, len(recent)-1; left < right; left, right =
		left+1, right-1 {

		recent[left],
			recent[right] =
			recent[right],
			recent[left]
	}

	return recent,
		nil
}

// RecentRaidBossIDsForRotation recupera apenas a quantidade
// necessária de histórico para cada raridade.
//
// Atualmente:
//
//	LEGENDARY -> últimos 6
//	MYTHIC    -> últimos 5
//	SACRED    -> últimos 2
func RecentRaidBossIDsForRotation(
	groupJID string,
) (
	[]string,
	error,
) {
	rarities :=
		[]Rarity{
			RarityLegendary,
			RarityMythic,
			RaritySacred,
		}

	recent :=
		make(
			[]string,
			0,
			RaidLegendaryRecentBlock+
				RaidMythicRecentBlock+
				RaidSacredRecentBlock,
		)

	for _, rarity := range rarities {
		limit :=
			RaidRotationRecentLimit(
				rarity,
			)

		if limit <= 0 {
			continue
		}

		ids, err :=
			RecentRaidBossIDs(
				groupJID,
				rarity,
				limit,
			)

		if err != nil {
			return nil,
				err
		}

		recent =
			append(
				recent,
				ids...,
			)
	}

	return recent,
		nil
}

// SelectRaidBossForGroup conecta o histórico persistido
// ao motor de rotação.
//
// Esta função apenas escolhe o Boss.
// O anúncio só deve ser registrado depois que a Raid
// realmente tiver sido criada.
func SelectRaidBossForGroup(
	catalog *RaidBossCatalog,
	groupJID string,
) (
	RaidBoss,
	error,
) {
	groupJID = strings.TrimSpace(groupJID)

	if groupJID == "" {
		return RaidBoss{},
			ErrInvalidRaidGroup
	}

	recent, err :=
		RecentRaidBossIDsForRotation(
			groupJID,
		)

	if err != nil {
		return RaidBoss{},
			err
	}

	return SelectRaidBoss(
		catalog,
		recent,
	)
}
