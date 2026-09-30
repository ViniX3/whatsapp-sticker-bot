package handler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"whatsapp-sticker-bot/internal/auth"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

const (
	raidResolutionPollInterval = 3 * time.Second

	raidUnavailableGroupRetry = 10 * time.Minute
)

var (
	raidUnavailableGroupMu sync.Mutex

	raidUnavailableGroups = make(
		map[string]time.Time,
	)
)

var raidResolutionWorkerOnce sync.Once

func StartRaidResolutionWorker(
	client *whatsmeow.Client,
) {
	if client == nil {
		return
	}

	raidResolutionWorkerOnce.Do(
		func() {
			go raidResolutionWorker(
				client,
			)

			logger.Success(
				"Worker automático de Raids iniciado",
			)
		},
	)
}

func raidResolutionWorker(
	client *whatsmeow.Client,
) {
	// Verifica imediatamente ao subir.
	processRaidResolutionCycle(
		client,
	)

	ticker :=
		time.NewTicker(
			raidResolutionPollInterval,
		)

	defer ticker.Stop()

	for range ticker.C {
		processRaidResolutionCycle(
			client,
		)
	}
}

func processRaidResolutionCycle(
	client *whatsmeow.Client,
) {
	now :=
		time.Now()

	due, err :=
		rpg.DueRaidSessions(
			now,
			20,
		)

	if err != nil {
		logger.Error(
			"Erro procurando Raids vencidas:",
			err,
		)

		return
	}

	for _, session := range due {

		roll, err :=
			rpg.RollRaidResolution()

		if err != nil {
			logger.Error(
				"Erro sorteando resultado da Raid:",
				session.ID,
				err,
			)

			continue
		}

		_, err =
			rpg.ResolveRaidSession(
				session.ID,
				now,
				roll,
			)

		if err != nil &&
			!errors.Is(
				err,
				rpg.ErrRaidAlreadyResolved,
			) {

			logger.Error(
				"Erro resolvendo Raid:",
				session.ID,
				err,
			)
		}
	}

	processPendingRaidRewards()

	sendPendingRaidResults(
		client,
	)

	ensureRaidWorldBosses(
		client,
		now,
	)
}

func sendPendingRaidResults(
	client *whatsmeow.Client,
) {
	results, err :=
		rpg.PendingRaidResolutionAnnouncements(
			20,
		)

	if err != nil {
		logger.Error(
			"Erro consultando anúncios pendentes de Raid:",
			err,
		)

		return
	}

	for _, result := range results {

		jid, err :=
			types.ParseJID(
				result.GroupJID,
			)

		if err != nil {
			logger.Error(
				"JID inválido em resultado de Raid:",
				result.GroupJID,
				err,
			)

			continue
		}

		message :=
			renderRaidResolutionResult(
				&result,
			)

		if result.Won {
			rewards, err :=
				rpg.RaidRewardsForRaid(
					result.RaidID,
				)

			if err != nil {
				logger.Error(
					"Erro carregando recompensas da Raid para anúncio:",
					result.RaidID,
					err,
				)

				continue
			}

			// Não anuncia vitória antes de todos os jogadores
			// terem suas recompensas persistidas.
			if len(rewards) !=
				result.ParticipantCount {

				logger.Warn(
					"Recompensas da Raid ainda incompletas:",
					"Raid:",
					result.RaidID,
					"Esperadas:",
					result.ParticipantCount,
					"Persistidas:",
					len(rewards),
				)

				continue
			}

			message +=
				renderRaidRewardSummary(
					rewards,
				)
		}

		if err :=
			whatsapp.SendText(
				client,
				jid,
				message,
			); err != nil {

			logger.Error(
				"Erro enviando resultado da Raid:",
				result.RaidID,
				err,
			)

			continue
		}

		if err :=
			rpg.MarkRaidResolutionAnnounced(
				result.RaidID,
				time.Now(),
			); err != nil {

			logger.Error(
				"Resultado enviado, mas falhou ao marcar anúncio:",
				result.RaidID,
				err,
			)

			continue
		}

		logger.Info(
			"Resultado de Raid anunciado:",
			"Raid:",
			result.RaidID,
			"Boss:",
			result.Boss.ID,
			"Vitória:",
			result.Won,
		)
	}
}

func renderRaidResolutionResult(
	result *rpg.RaidResolutionResult,
) string {
	if result == nil {
		return "⚔️ A Raid foi encerrada."
	}

	if result.ParticipantCount == 0 {
		return fmt.Sprintf(
			"💀 *RAID ENCERRADA*\n\n"+
				"%s *%s*\n\n"+
				"👥 Nenhum aventureiro entrou na batalha.\n"+
				"⚔️ Poder coletivo: *0 PC*\n"+
				"💀 Poder do Boss: *%s PC*\n\n"+
				"☠️ O Raid Boss venceu sem encontrar resistência.",
			raidRarityIcon(
				result.Boss.Rarity,
			),
			result.Boss.Name,
			formatRPGNumber(
				result.Boss.Power,
			),
		)
	}

	if result.Won {
		return fmt.Sprintf(
			"🏆 *RAID CONCLUÍDA — VITÓRIA!* 🏆\n\n"+
				"%s *%s foi derrotado!*\n\n"+
				"👥 Aventureiros: *%d*\n"+
				"⚔️ Poder coletivo: *%s PC*\n"+
				"💀 Poder do Boss: *%s PC*\n"+
				"🎯 Chance de vitória: *%d%%*\n"+
				"🎲 Resultado: *%d/100*\n\n"+
				"✨ O grupo triunfou sobre o Raid Boss!",
			raidRarityIcon(
				result.Boss.Rarity,
			),
			result.Boss.Name,
			result.ParticipantCount,
			formatRPGNumber(
				result.TotalPower,
			),
			formatRPGNumber(
				result.Boss.Power,
			),
			result.SuccessChance,
			result.Roll,
		)
	}

	return fmt.Sprintf(
		"💀 *RAID CONCLUÍDA — DERROTA*\n\n"+
			"%s *%s resistiu ao ataque!*\n\n"+
			"👥 Aventureiros: *%d*\n"+
			"⚔️ Poder coletivo: *%s PC*\n"+
			"💀 Poder do Boss: *%s PC*\n"+
			"🎯 Chance de vitória: *%d%%*\n"+
			"🎲 Resultado: *%d/100*\n\n"+
			"🌑 O Raid Boss permaneceu de pé.",
		raidRarityIcon(
			result.Boss.Rarity,
		),
		result.Boss.Name,
		result.ParticipantCount,
		formatRPGNumber(
			result.TotalPower,
		),
		formatRPGNumber(
			result.Boss.Power,
		),
		result.SuccessChance,
		result.Roll,
	)
}

func ensureRaidWorldBosses(
	client *whatsmeow.Client,
	now time.Time,
) {
	bossCatalog, err :=
		getRaidBossCatalog()

	if err != nil {
		logger.Error(
			"Erro carregando catálogo de Raid Bosses automáticos:",
			err,
		)

		return
	}

	itemCatalog,
		_,
		err :=
		getRPGCatalogs()

	if err != nil {
		logger.Error(
			"Erro carregando catálogo RPG para Raid automática:",
			err,
		)

		return
	}

	for _, groupJID := range auth.AllowedGroupIDs() {

		hasPlayers, err :=
			rpg.RaidGroupHasPlayers(
				groupJID,
			)

		if err != nil {
			logger.Error(
				"Erro verificando jogadores do grupo para Raid:",
				groupJID,
				err,
			)

			continue
		}

		if !hasPlayers {
			continue
		}

		canSpawn, err :=
			rpg.RaidCanSpawnForGroup(
				groupJID,
				now,
			)

		if err != nil {
			logger.Error(
				"Erro verificando respawn de Raid:",
				groupJID,
				err,
			)

			continue
		}

		if !canSpawn {
			continue
		}

		jid,
			reachable :=
			raidGroupReachable(
				client,
				groupJID,
				now,
			)

		if !reachable {
			continue
		}

		opened, err :=
			rpg.OpenDormantRaidForGroup(
				bossCatalog,
				itemCatalog,
				groupJID,
				now,
			)

		if err != nil {

			if errors.Is(
				err,
				rpg.ErrRaidActiveExists,
			) {
				continue
			}

			logger.Error(
				"Erro criando Raid Boss automático:",
				groupJID,
				err,
			)

			continue
		}

		lobby, err :=
			rpg.LoadActiveRaidLobby(
				groupJID,
			)

		if err != nil {
			logger.Error(
				"Raid criada mas lobby não pôde ser carregado:",
				groupJID,
				err,
			)

			continue
		}

		message :=
			renderRaidLobby(
				lobby,
				now,
				true,
			)

		if !sendRaidBossImage(
			client,
			jid,
			lobby,
			message,
		) {

			if err :=
				whatsapp.SendText(
					client,
					jid,
					message,
				); err != nil {

				logger.Error(
					"Erro enviando anúncio automático de Raid:",
					groupJID,
					err,
				)

				continue
			}
		}

		logger.Info(
			"Raid Boss automático surgiu:",
			"Grupo:",
			groupJID,
			"Boss:",
			opened.Boss.ID,
			"Poder:",
			opened.Boss.Power,
		)
	}
}

func raidGroupReachable(
	client *whatsmeow.Client,
	groupJID string,
	now time.Time,
) (
	types.JID,
	bool,
) {
	jid, err :=
		types.ParseJID(
			groupJID,
		)

	if err != nil {
		logger.Error(
			"JID inválido na whitelist de Raid:",
			groupJID,
			err,
		)

		return types.JID{},
			false
	}

	raidUnavailableGroupMu.Lock()

	retryAt,
		blocked :=
		raidUnavailableGroups[groupJID]

	raidUnavailableGroupMu.Unlock()

	if blocked &&
		now.Before(
			retryAt,
		) {

		return jid,
			false
	}

	ctx,
		cancel :=
		context.WithTimeout(
			context.Background(),
			10*time.Second,
		)

	defer cancel()

	_,
		err =
		client.GetGroupInfo(
			ctx,
			jid,
		)

	if err != nil {

		raidUnavailableGroupMu.Lock()

		raidUnavailableGroups[groupJID] =
			now.Add(
				raidUnavailableGroupRetry,
			)

		raidUnavailableGroupMu.Unlock()

		logger.Warn(
			"Grupo ignorado para Raid automática; bot sem acesso:",
			groupJID,
			err,
		)

		return jid,
			false
	}

	raidUnavailableGroupMu.Lock()

	delete(
		raidUnavailableGroups,
		groupJID,
	)

	raidUnavailableGroupMu.Unlock()

	return jid,
		true
}
