package handler

import (
	"fmt"
	"strings"

	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/rpg"
)

func processPendingRaidRewards() {
	raidIDs, err :=
		rpg.PendingRaidRewardIDs(
			20,
		)

	if err != nil {
		logger.Error(
			"Erro consultando recompensas pendentes de Raid:",
			err,
		)

		return
	}

	if len(raidIDs) == 0 {
		return
	}

	catalog,
		_,
		err :=
		getRPGCatalogs()

	if err != nil {
		logger.Error(
			"Erro carregando catálogo para recompensas Raid:",
			err,
		)

		return
	}

	for _, raidID := range raidIDs {

		rewards, err :=
			rpg.ApplyRaidRewards(
				raidID,
				catalog,
			)

		if err != nil {
			logger.Error(
				"Erro aplicando recompensas da Raid:",
				raidID,
				err,
			)

			continue
		}

		logger.Info(
			"Recompensas da Raid processadas:",
			"Raid:",
			raidID,
			"Jogadores:",
			len(rewards),
		)
	}
}

func renderRaidRewardSummary(
	rewards []rpg.RaidPlayerReward,
) string {
	if len(rewards) == 0 {
		return ""
	}

	var builder strings.Builder

	builder.WriteString(
		"\n\n🎁 *RECOMPENSAS DA RAID*",
	)

	for _, reward := range rewards {

		name :=
			strings.TrimSpace(
				reward.Name,
			)

		if name == "" {
			name =
				reward.JID
		}

		fmt.Fprintf(
			&builder,
			"\n\n👤 *%s*\n"+
				"💰 %s Gold\n"+
				"💎 %s Cristais",
			name,
			formatRPGNumber(
				reward.Gold,
			),
			formatRPGNumber(
				reward.MagicCrystals,
			),
		)

		if reward.StellarStones > 0 {
			fmt.Fprintf(
				&builder,
				"\n🌠 %d Pedra(s) Estelar(es)",
				reward.StellarStones,
			)
		}

		if strings.TrimSpace(
			reward.ItemName,
		) != "" {

			fmt.Fprintf(
				&builder,
				"\n🎁 *%s*",
				reward.ItemName,
			)
		}
	}

	return builder.String()
}
