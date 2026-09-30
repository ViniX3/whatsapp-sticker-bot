package rpg

import (
	"fmt"
	"strings"
	"time"
)

type OpenRaidResult struct {
	Session *RaidSession

	Boss RaidBoss
}

// OpenRaidForGroup coordena a abertura de uma nova Raid.
//
// Ordem:
//  1. escolhe o Boss respeitando a rotação do grupo;
//  2. cria a sessão persistente;
//  3. registra o Boss no histórico.
//
// O histórico só é consumido depois que a sessão existe.
func OpenRaidForGroup(
	catalog *RaidBossCatalog,
	groupJID string,
	now time.Time,
) (
	*OpenRaidResult,
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

	if catalog == nil {
		return nil,
			fmt.Errorf(
				"%w: catálogo nil",
				ErrInvalidRaidBossCatalog,
			)
	}

	if now.IsZero() {
		now = time.Now()
	}

	boss, err :=
		SelectRaidBossForGroup(
			catalog,
			groupJID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro selecionando Raid Boss: %w",
				err,
			)
	}

	session, err :=
		CreateRaidSession(
			groupJID,
			boss,
			now,
		)

	if err != nil {
		return nil,
			err
	}

	_, err =
		RecordRaidAnnouncement(
			groupJID,
			boss,
			now,
		)

	if err != nil {
		closeErr :=
			CloseActiveRaidSession(
				groupJID,
				RaidSessionCancelled,
				now,
			)

		if closeErr != nil {
			return nil,
				fmt.Errorf(
					"erro registrando histórico da Raid: %v; "+
						"erro cancelando sessão criada: %v",
					err,
					closeErr,
				)
		}

		return nil,
			fmt.Errorf(
				"erro registrando histórico da Raid: %w",
				err,
			)
	}

	return &OpenRaidResult{
			Session: session,

			Boss: boss,
		},
		nil
}

// RaidParticipantFromEquipmentSummary converte o estado atual
// do equipamento em um snapshot imutável para a Raid.
func RaidParticipantFromEquipmentSummary(
	jid string,
	name string,
	summary *EquipmentSummary,
) (
	RaidParticipant,
	error,
) {
	jid =
		strings.TrimSpace(
			jid,
		)

	name =
		strings.TrimSpace(
			name,
		)

	if jid == "" ||
		summary == nil ||
		summary.CombatPower <= 0 {

		return RaidParticipant{},
			ErrRaidInvalidParticipant
	}

	bonuses :=
		RaidBonusesFromSets(
			summary.ActiveSetBonuses,
		)

	return RaidParticipant{
			JID: jid,

			Name: name,

			CombatPower: summary.CombatPower,

			EffectivePower: RaidEffectiveBossPower(
				summary.CombatPower,
				bonuses.BossDamagePercent,
			),

			Bonuses: bonuses,
		},
		nil
}

// PrepareRaidParticipant consulta os equipamentos reais
// que o jogador possui no momento da entrada.
func PrepareRaidParticipant(
	groupJID string,
	jid string,
	name string,
	itemCatalog *Catalog,
) (
	RaidParticipant,
	error,
) {
	groupJID =
		strings.TrimSpace(
			groupJID,
		)

	jid =
		strings.TrimSpace(
			jid,
		)

	if groupJID == "" {
		return RaidParticipant{},
			ErrInvalidRaidGroup
	}

	if jid == "" {
		return RaidParticipant{},
			ErrRaidInvalidParticipant
	}

	if itemCatalog == nil {
		return RaidParticipant{},
			fmt.Errorf(
				"catálogo RPG não inicializado",
			)
	}

	summary, err :=
		GetEquipmentSummary(
			groupJID,
			jid,
			itemCatalog,
		)

	if err != nil {
		return RaidParticipant{},
			fmt.Errorf(
				"erro consultando equipamentos para Raid: %w",
				err,
			)
	}

	participant, err :=
		RaidParticipantFromEquipmentSummary(
			jid,
			name,
			summary,
		)

	if err != nil {
		return RaidParticipant{},
			err
	}

	return participant,
		nil
}

// JoinRaidWithCurrentEquipment é a operação usada futuramente
// pelo comando !boss entrar.
func JoinRaidWithCurrentEquipment(
	groupJID string,
	jid string,
	name string,
	itemCatalog *Catalog,
	now time.Time,
) (
	RaidParticipant,
	error,
) {
	participant, err :=
		PrepareRaidParticipant(
			groupJID,
			jid,
			name,
			itemCatalog,
		)

	if err != nil {
		return RaidParticipant{},
			err
	}

	if err :=
		JoinActiveRaidSession(
			groupJID,
			participant,
			now,
		); err != nil {

		return RaidParticipant{},
			err
	}

	return participant,
		nil
}
