package handler

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	raidBossCatalogOnce sync.Once

	cachedRaidBossCatalog *rpg.RaidBossCatalog

	cachedRaidBossCatalogErr error
)

func getRaidBossCatalog() (
	*rpg.RaidBossCatalog,
	error,
) {
	raidBossCatalogOnce.Do(
		func() {
			cachedRaidBossCatalog,
				cachedRaidBossCatalogErr =
				rpg.LoadRaidBossCatalog()
		},
	)

	return cachedRaidBossCatalog,
		cachedRaidBossCatalogErr
}

func handleRPGRaid(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) {
	action, ok :=
		parseRaidCommand(
			text,
		)

	if !ok {
		sendRPGRaidUsage(
			client,
			msgEvent,
		)

		return
	}

	switch action {

	case raidActionStatus:

		sendActiveRaidStatus(
			client,
			msgEvent,
			false,
		)

	case raidActionStart:

		handleRPGRaidStart(
			client,
			msgEvent,
		)

	case raidActionJoin:

		handleRPGRaidJoin(
			client,
			msgEvent,
		)
	}
}

func handleRPGRaidStart(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"🌑 Os *Raid Bosses agora surgem automaticamente*.\n\n"+
			"Use *!boss info* para consultar o Boss atual.\n"+
			"Use *!boss entrar* para enfrentá-lo.",
	)
}

func handleRPGRaidJoin(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	name :=
		strings.TrimSpace(
			msgEvent.Info.PushName,
		)

	if name == "" {
		name =
			msgEvent.Info.Sender.User
	}

	itemCatalog,
		_,
		err :=
		getRPGCatalogs()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo RPG para Raid",
			err,
		)

		return
	}

	if _, err :=
		rpg.GetOrCreatePlayer(
			groupJID,
			jid,
		); err != nil {

		sendRPGPlayerError(
			client,
			msgEvent,
			err,
		)

		return
	}

	participant, err :=
		rpg.JoinRaidWithCurrentEquipment(
			groupJID,
			jid,
			name,
			itemCatalog,
			time.Now(),
		)

	if err != nil {

		switch {

		case errors.Is(
			err,
			rpg.ErrRaidSessionNotFound,
		):

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"🌑 Não existe nenhuma *Raid ativa* neste momento.",
			)

		case errors.Is(
			err,
			rpg.ErrRaidAlreadyJoined,
		):

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"⚔️ Você já está participando desta Raid.",
			)

		case errors.Is(
			err,
			rpg.ErrRaidLobbyFull,
		):

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"👥 A Raid já atingiu o limite de *10 participantes*.",
			)

		case errors.Is(
			err,
			rpg.ErrRaidLobbyClosed,
		):

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"⏳ O período de entrada desta Raid já terminou.",
			)

		case errors.Is(
			err,
			rpg.ErrRaidInvalidParticipant,
		):

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Você ainda não possui Poder de Combate suficiente para entrar.\n\n"+
					"Confira seus equipamentos com *!equipamentos*.",
			)

		default:

			sendRPGInternalError(
				client,
				msgEvent,
				"entrando na Raid",
				err,
			)
		}

		return
	}

	lobby, err :=
		rpg.LoadActiveRaidLobby(
			groupJID,
		)

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"recarregando lobby da Raid",
			err,
		)

		return
	}

	remaining :=
		formatRaidRemaining(
			lobby.StartsAt,
			time.Now(),
		)

	response :=
		fmt.Sprintf(
			"⚔️ *%s entrou na Raid!*\n\n"+
				"⚡ PC: *%s*\n"+
				"🔥 Poder contra Boss: *%s*\n"+
				"👥 Participantes: *%d/%d*\n"+
				"⚔️ Poder coletivo: *%s*\n"+
				"⏳ Batalha em: *%s*",
			participant.Name,
			formatRPGNumber(
				participant.CombatPower,
			),
			formatRPGNumber(
				participant.EffectivePower,
			),
			len(
				lobby.Participants,
			),
			rpg.RaidMaxParticipants,
			formatRPGNumber(
				lobby.TotalPower(),
			),
			remaining,
		)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}

func sendActiveRaidStatus(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	announcement bool,
) {
	groupJID :=
		msgEvent.Info.Chat.String()

	lobby, err :=
		rpg.LoadActiveRaidLobby(
			groupJID,
		)

	if err != nil {

		if errors.Is(
			err,
			rpg.ErrRaidSessionNotFound,
		) {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"🌑 Nenhum *Raid Boss* está presente neste momento.\n\n"+
					"Outro surgirá automaticamente em breve.",
			)

			return
		}

		sendRPGInternalError(
			client,
			msgEvent,
			"consultando Raid ativa",
			err,
		)

		return
	}

	message :=
		renderRaidLobby(
			lobby,
			time.Now(),
			announcement,
		)

	if sendRaidBossAnnouncement(
		client,
		msgEvent,
		lobby,
		message,
	) {

		return
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		message,
	)
}

func renderRaidLobby(
	lobby *rpg.RaidLobby,
	now time.Time,
	announcement bool,
) string {
	if lobby == nil {
		return "🌑 Nenhum Raid Boss disponível."
	}

	var builder strings.Builder

	description :=
		strings.TrimSpace(
			lobby.Boss.Description,
		)

	if announcement {

		if description != "" {
			fmt.Fprintf(
				&builder,
				"⚡ _%s_\n\n",
				description,
			)
		}

		fmt.Fprintf(
			&builder,
			"🔥 *O %s SURGIU!*\n\n"+
				"Use:\n"+
				"*!boss entrar*\n"+
				"*!boss info*",
			strings.ToUpper(
				lobby.Boss.Name,
			),
		)

		return builder.String()
	}

	fmt.Fprintf(
		&builder,
		"%s *%s*\n"+
			"🏷️ %s\n"+
			"💀 Poder: *%s PC*\n",
		raidRarityIcon(
			lobby.Boss.Rarity,
		),
		lobby.Boss.Name,
		raidRarityName(
			lobby.Boss.Rarity,
		),
		formatRPGNumber(
			lobby.Boss.Power,
		),
	)

	if description != "" {
		fmt.Fprintf(
			&builder,
			"\n📜 _%s_\n",
			description,
		)
	}

	participantCount :=
		len(
			lobby.Participants,
		)

	if participantCount == 0 {
		builder.WriteString(
			"\n🌑 O Boss permanece no mundo, aguardando um desafiante.\n\n" +
				"⚔️ Use *!boss entrar* para iniciar o confronto.",
		)

		return builder.String()
	}

	fmt.Fprintf(
		&builder,
		"\n👥 Guerreiros: *%d/%d*\n"+
			"⏳ Reforços podem entrar por: *%s*",
		participantCount,
		rpg.RaidMaxParticipants,
		formatRaidRemaining(
			lobby.StartsAt,
			now,
		),
	)

	builder.WriteString(
		"\n\n👤 *GRUPO DA RAID*",
	)

	for index, participant := range lobby.Participants {

		name :=
			strings.TrimSpace(
				participant.Name,
			)

		if name == "" {
			name =
				participant.JID
		}

		fmt.Fprintf(
			&builder,
			"\n%d. %s",
			index+1,
			name,
		)
	}

	if lobby.IsOpen(
		now,
	) {
		builder.WriteString(
			"\n\n⚔️ Use *!boss entrar* para participar.",
		)
	} else {
		builder.WriteString(
			"\n\n🔒 *As inscrições foram encerradas.*",
		)
	}

	return builder.String()
}

func formatRaidRemaining(
	startsAt time.Time,
	now time.Time,
) string {
	if !now.Before(
		startsAt,
	) {
		return "encerrado"
	}

	remaining :=
		startsAt.Sub(
			now,
		)

	seconds :=
		int64(
			(remaining +
				time.Second -
				1) /
				time.Second,
		)

	minutes :=
		seconds / 60

	seconds =
		seconds % 60

	return fmt.Sprintf(
		"%dm %02ds",
		minutes,
		seconds,
	)
}

func raidRarityIcon(
	rarity rpg.Rarity,
) string {
	switch rarity {

	case rpg.RarityLegendary:
		return "🔴"

	case rpg.RarityMythic:
		return "🟠"

	case rpg.RaritySacred:
		return "🌟"
	}

	return "⚔️"
}

func raidRarityName(
	rarity rpg.Rarity,
) string {
	switch rarity {

	case rpg.RarityLegendary:
		return "LENDÁRIO"

	case rpg.RarityMythic:
		return "MÍTICO"

	case rpg.RaritySacred:
		return "SAGRADO"
	}

	return string(
		rarity,
	)
}

func sendRPGRaidUsage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"⚔️ *RAID BOSS*\n\n"+
			"*!boss info* — informações do Boss atual\n"+
			"*!boss entrar* — entrar na batalha\n\n"+
			"Aliases: *!raid* e *!raide*",
	)
}
