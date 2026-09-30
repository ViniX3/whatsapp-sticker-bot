package handler

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"whatsapp-sticker-bot/internal/database"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// handleGatheringCommand trata:
//
//	!coletar
//	!coletar floresta
//	!coletar pedreira
//	!coletar mina
//
// Sem região informada, o próprio RPG escolhe uma
// região aleatoriamente.
func handleGatheringCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts := strings.Fields(text)

	if len(parts) == 0 ||
		strings.ToLower(parts[0]) != "!coletar" {

		return false
	}

	if !requireGoldGroup(
		client,
		msgEvent,
		"!coletar",
	) {
		return true
	}

	if len(parts) > 2 {
		sendGatheringUsage(
			client,
			msgEvent,
		)

		return true
	}

	var region rpg.GatheringRegion

	if len(parts) == 2 {
		var ok bool

		region, ok =
			rpg.ParseGatheringRegion(
				parts[1],
			)

		if !ok {
			sendGatheringUsage(
				client,
				msgEvent,
			)

			return true
		}
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	itemCatalog, materialCatalog, err :=
		getRPGCatalogs()

	if err != nil {
		logger.Error(
			"Erro carregando catálogos para !coletar:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível carregar os recursos do RPG.",
		)

		return true
	}

	// ======================================================
	// CARTEIRA / MIGRAÇÃO RPG
	// ======================================================
	//
	// !coletar não cria Gold automaticamente.
	//
	// Usuário novo precisa iniciar sua jornada com
	// !gold ou !rpg.
	//
	// Usuário antigo que já possui carteira, mas ainda
	// não possui RPG, recebe apenas o Set do Recruta.

	wallet, err :=
		database.GetWallet(
			groupJID,
			jid,
		)

	if err != nil {
		logger.Error(
			"Erro consultando carteira para !coletar:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível verificar sua carteira agora.",
		)

		return true
	}

	if wallet == nil {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"📜 Você ainda não iniciou sua jornada neste grupo.\n\nUse *!gold* ou *!rpg* primeiro.",
		)

		return true
	}

	_, err =
		rpg.ClaimStarterSet(
			groupJID,
			jid,
			itemCatalog,
		)

	if err != nil &&
		!errors.Is(
			err,
			rpg.ErrStarterAlreadyClaimed,
		) {

		logger.Error(
			"Erro preparando RPG para !coletar:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível preparar sua jornada RPG para a coleta.",
		)

		return true
	}

	// ======================================================
	// COLETA
	// ======================================================

	result, err :=
		rpg.Gather(
			groupJID,
			jid,
			region,
			materialCatalog,
		)

	if err != nil {
		var cooldownErr *rpg.GatheringCooldownError

		switch {
		case errors.As(
			err,
			&cooldownErr,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				fmt.Sprintf(
					"⏳ Você ainda está descansando da última coleta.\n\nTente novamente em *%s*.",
					formatGatheringCooldown(
						cooldownErr.Remaining,
					),
				),
			)

		case errors.Is(
			err,
			rpg.ErrInvalidGatheringRegion,
		):
			sendGatheringUsage(
				client,
				msgEvent,
			)

		default:
			logger.Error(
				"Erro executando !coletar:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível concluir a coleta agora.",
			)
		}

		return true
	}

	// ======================================================
	// RESPOSTA
	// ======================================================

	var response strings.Builder

	fmt.Fprintf(
		&response,
		"%s *COLETA CONCLUÍDA — %s*\n\n",
		result.Region.Icon(),
		result.Region.Name(),
	)

	response.WriteString(
		"🎒 *Recursos encontrados:*\n",
	)

	for _, drop :=
		range result.Drops {

		fmt.Fprintf(
			&response,
			"%s *%s* ×%d\n",
			rpgRarityIcon(
				drop.Material.Rarity,
			),
			drop.Material.Name,
			drop.Quantity,
		)
	}

	fmt.Fprintf(
		&response,
		"\n📦 Total coletado: *%d recursos*",
		result.TotalItems,
	)

	if result.BonusRoll {
		response.WriteString(
			"\n🍀 *Sorte!* Você encontrou recursos extras nesta expedição.",
		)
	}

	response.WriteString(
		"\n\n⏳ Você poderá coletar novamente em *30 minutos*.",
	)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response.String(),
	)

	logger.Info(
		"Coleta RPG concluída:",
		"JID:",
		jid,
		"Região:",
		result.Region,
		"Rolagens:",
		result.Rolls,
		"Total:",
		result.TotalItems,
		"Bonus:",
		result.BonusRoll,
	)

	return true
}

func sendGatheringUsage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"🌿 *COLETA DE RECURSOS*\n\n"+
			"Use:\n"+
			"*!coletar*\n"+
			"Região aleatória.\n\n"+
			"Ou escolha uma região:\n\n"+
			"🌲 *!coletar floresta*\n"+
			"🪨 *!coletar pedreira*\n"+
			"⛏️ *!coletar mina*\n\n"+
			"⏳ Cada aventureiro pode coletar uma vez a cada *30 minutos*.",
	)
}

func formatGatheringCooldown(
	duration time.Duration,
) string {
	if duration <= 0 {
		return "alguns segundos"
	}

	duration =
		duration.Round(
			time.Second,
		)

	minutes :=
		int(
			duration /
				time.Minute,
		)

	seconds :=
		int(
			(duration %
				time.Minute) /
				time.Second,
		)

	if minutes > 0 {
		return fmt.Sprintf(
			"%dm %02ds",
			minutes,
			seconds,
		)
	}

	if seconds < 1 {
		seconds = 1
	}

	return fmt.Sprintf(
		"%ds",
		seconds,
	)
}
