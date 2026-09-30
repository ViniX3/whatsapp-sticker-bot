package rpg

import (
	"fmt"
	"sort"
	"strings"

	"whatsapp-sticker-bot/internal/database"
)

const raidScalingReferencePlayers = 5

func RaidScalePowerFromReference(
	basePower int,
	rarity Rarity,
	referencePower int,
) int {
	if basePower <= 0 {
		return 0
	}

	if referencePower <= 0 {
		return basePower
	}

	multiplier := 5

	switch rarity {

	case RarityLegendary:
		multiplier = 5

	case RarityMythic:
		multiplier = 6

	case RaritySacred:
		multiplier = 8
	}

	scaled :=
		int64(referencePower) *
			int64(multiplier)

	if scaled <
		int64(basePower) {

		return basePower
	}

	return int(scaled)
}

func RaidScaledBossPower(
	groupJID string,
	boss RaidBoss,
	catalog *Catalog,
) (
	int,
	error,
) {
	groupJID =
		strings.TrimSpace(
			groupJID,
		)

	if groupJID == "" {
		return 0,
			ErrInvalidRaidGroup
	}

	if catalog == nil {
		return 0,
			fmt.Errorf(
				"catálogo RPG não inicializado",
			)
	}

	if err :=
		ensureSchema(); err != nil {

		return 0, err
	}

	rows, err :=
		database.DB.Query(
			`
			SELECT jid
			FROM rpg_players
			WHERE group_jid = ?
			`,
			groupJID,
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro consultando jogadores para escala da Raid: %w",
				err,
			)
	}

	jids :=
		make(
			[]string,
			0,
		)

	for rows.Next() {
		var jid string

		if err :=
			rows.Scan(
				&jid,
			); err != nil {

			rows.Close()

			return 0,
				fmt.Errorf(
					"erro lendo jogador para escala da Raid: %w",
					err,
				)
		}

		jids =
			append(
				jids,
				jid,
			)
	}

	if err :=
		rows.Err(); err != nil {

		rows.Close()

		return 0, err
	}

	rows.Close()

	powers :=
		make(
			[]int,
			0,
			len(jids),
		)

	for _, jid := range jids {

		summary, err :=
			GetEquipmentSummary(
				groupJID,
				jid,
				catalog,
			)

		if err != nil {
			return 0,
				fmt.Errorf(
					"erro consultando PC de %s para Raid: %w",
					jid,
					err,
				)
		}

		if summary == nil ||
			summary.CombatPower <= 0 {

			continue
		}

		powers =
			append(
				powers,
				summary.CombatPower,
			)
	}

	if len(powers) == 0 {
		return boss.Power,
			nil
	}

	sort.Sort(
		sort.Reverse(
			sort.IntSlice(
				powers,
			),
		),
	)

	limit :=
		len(powers)

	if limit >
		raidScalingReferencePlayers {

		limit =
			raidScalingReferencePlayers
	}

	total :=
		int64(0)

	for index := 0; index < limit; index++ {

		total +=
			int64(
				powers[index],
			)
	}

	reference :=
		int(
			total /
				int64(limit),
		)

	return RaidScalePowerFromReference(
		boss.Power,
		boss.Rarity,
		reference,
	), nil
}
