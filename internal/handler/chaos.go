package handler

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"whatsapp-sticker-bot/internal/auth"
	"whatsapp-sticker-bot/internal/chaos"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var chaosSchedulerOnce sync.Once

// StartChaosScheduler inicia o scheduler global exatamente
// uma vez durante a vida do processo.
func StartChaosScheduler(
	client *whatsmeow.Client,
) {
	chaosSchedulerOnce.Do(
		func() {
			go func() {
				logger.Success(
					"Scheduler do Presságio do Caos iniciado",
				)

				err :=
					chaos.RunScheduler(
						func(event chaos.Event) {
							logger.Success(
								fmt.Sprintf(
									"Presságio do Caos iniciado: evento=%d bônus=+%d%% duração=%s",
									event.ID,
									event.BonusPercent,
									formatChaosDuration(
										time.Until(
											event.EndsAt,
										),
									),
								),
							)

							announceChaosStart(
								client,
								event,
							)
						},
						func(event chaos.Event) {
							logger.Info(
								fmt.Sprintf(
									"Presságio do Caos encerrado: evento=%d",
									event.ID,
								),
							)

							announceChaosEnd(
								client,
								event,
							)
						},
					)

				if err != nil {
					logger.Error(
						"Scheduler do Presságio do Caos encerrado por erro:",
						err,
					)
				}
			}()
		},
	)
}

func handleChaosCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts :=
		strings.Fields(text)

	if len(parts) == 0 ||
		!strings.EqualFold(
			parts[0],
			"!pressagio",
		) {

		return false
	}

	if !requireGoldGroup(
		client,
		msgEvent,
		"!pressagio",
	) {
		return true
	}

	if len(parts) != 1 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"☄️ Uso correto: *!pressagio*",
		)

		return true
	}

	state :=
		chaos.Snapshot()

	if state.Active {
		remaining :=
			time.Until(
				state.Event.EndsAt,
			)

		if remaining < 0 {
			remaining = 0
		}

		response :=
			fmt.Sprintf(
				"☄️ *PRESSÁGIO DO CAOS ATIVO* ☄️\n\n"+
					"🌌 Uma ruptura cobre o reino.\n\n"+
					"🔥 Bônus atual: *+%d%%*\n"+
					"⏳ Tempo restante: *%s*\n\n"+
					"🎲 Jogos afetados:\n"+
					"*!bet • !slots • !caraoucoroa • !quiz • !forca • !loteria*\n\n"+
					"🍀 *!sorte não recebe o bônus.*",
				state.Event.BonusPercent,
				formatChaosDuration(
					remaining,
				),
			)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			response,
		)

		return true
	}

	response :=
		"🌑 *O REINO ESTÁ CALMO*\n\n" +
			"Nenhum Presságio do Caos está ativo neste momento."

	if !state.NextStartsAt.IsZero() {
		remaining :=
			time.Until(
				state.NextStartsAt,
			)

		if remaining < 0 {
			remaining = 0
		}

		response +=
			fmt.Sprintf(
				"\n\n🔮 Os oráculos sentem uma nova ruptura em aproximadamente *%s*.",
				formatChaosDuration(
					remaining,
				),
			)
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)

	return true
}

func announceChaosStart(
	client *whatsmeow.Client,
	event chaos.Event,
) {
	duration :=
		time.Until(
			event.EndsAt,
		)

	if duration < 0 {
		duration = 0
	}

	message :=
		fmt.Sprintf(
			"☄️ *PRESSÁGIO DO CAOS* ☄️\n\n"+
				"O céu sobre o reino se rompeu...\n\n"+
				"🔥 As recompensas dos jogos foram amplificadas!\n"+
				"🍀 Bônus do Caos: *+%d%%*\n"+
				"⏳ Duração: *%s*\n\n"+
				"🎲 Durante a ruptura:\n"+
				"*!bet • !slots • !caraoucoroa • !quiz • !forca • !loteria*\n\n"+
				"⚠️ *!sorte não é afetado.*\n\n"+
				"Aproveitem enquanto a ruptura permanece aberta...",
			event.BonusPercent,
			formatChaosDuration(
				duration,
			),
		)

	announceChaosToAllowedGroups(
		client,
		message,
	)
}

func announceChaosEnd(
	client *whatsmeow.Client,
	event chaos.Event,
) {
	message :=
		"🌑 *O PRESSÁGIO DO CAOS TERMINOU*\n\n" +
			"A ruptura dimensional se fechou.\n\n" +
			"🔥 As recompensas voltaram aos valores normais.\n\n" +
			"Os oráculos aguardam o próximo sinal..."

	announceChaosToAllowedGroups(
		client,
		message,
	)
}

func announceChaosToAllowedGroups(
	client *whatsmeow.Client,
	message string,
) {
	groups :=
		auth.AllowedGroupIDs()

	for _, group := range groups {

		jid, err :=
			types.ParseJID(
				group,
			)

		if err != nil {
			logger.Warn(
				"JID inválido na whitelist durante anúncio do Presságio:",
				group,
				err,
			)

			continue
		}

		if err :=
			whatsapp.SendText(
				client,
				jid,
				message,
			); err != nil {

			logger.Warn(
				"Falha ao anunciar Presságio no grupo:",
				group,
				err,
			)
		}
	}
}

func formatChaosDuration(
	duration time.Duration,
) string {
	if duration <= 0 {
		return "menos de 1 minuto"
	}

	minutes :=
		int(
			duration.Round(
				time.Minute,
			) /
				time.Minute,
		)

	if minutes < 1 {
		minutes = 1
	}

	hours :=
		minutes / 60

	remainingMinutes :=
		minutes % 60

	switch {
	case hours > 0 &&
		remainingMinutes > 0:

		return fmt.Sprintf(
			"%dh %dmin",
			hours,
			remainingMinutes,
		)

	case hours > 0:
		return fmt.Sprintf(
			"%dh",
			hours,
		)

	default:
		return fmt.Sprintf(
			"%d min",
			minutes,
		)
	}
}

func formatChaosGoldBonus(
	bonus int,
	percent int,
) string {
	if bonus <= 0 ||
		percent <= 0 {

		return ""
	}

	return fmt.Sprintf(
		"\n☄️ *Presságio do Caos:* +%d Gold (*+%d%%*)",
		bonus,
		percent,
	)
}

// applyChaosGameReward centraliza o bônus usado pelos
// jogos controlados diretamente pelos handlers.
//
// Retorna:
//
//	recompensa final
//	bônus adicional
//	percentual do Presságio
func applyChaosGameReward(
	baseReward int,
) (
	int,
	int,
	int,
) {
	total,
		bonus,
		percent,
		active :=
		chaos.ApplyGoldReward(
			baseReward,
		)

	if !active {
		return baseReward,
			0,
			0
	}

	return total,
		bonus,
		percent
}
