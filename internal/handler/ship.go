package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type shipPlayer struct {
	JID  types.JID
	Name string
}

func handleShipCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts := strings.Fields(text)

	if len(parts) == 0 ||
		!strings.EqualFold(
			parts[0],
			"!ship",
		) {
		return false
	}

	if !msgEvent.Info.IsGroup {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ O comando *!ship* só pode ser usado em grupos.",
		)

		return true
	}

	groupInfo, err :=
		client.GetGroupInfo(
			context.Background(),
			msgEvent.Info.Chat,
		)

	if err != nil {
		logger.Error(
			"Erro ao consultar participantes para !ship:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível consultar os participantes do grupo agora.",
		)

		return true
	}

	mentioned :=
		shipMentionedJIDs(
			msgEvent,
		)

	var first shipPlayer
	var second shipPlayer

	switch len(mentioned) {

	// ======================================================
	// NENHUMA MARCAÇÃO
	//
	// Sorteia duas pessoas aleatórias do grupo.
	// ======================================================

	case 0:
		if len(parts) != 1 {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"💘 Use *!ship*, *!ship @pessoa* ou *!ship @pessoa1 @pessoa2*.",
			)

			return true
		}

		first, second, err =
			randomShipPair(
				client,
				groupInfo.Participants,
			)

		if err != nil {
			logger.Error(
				"Erro ao sortear participantes do !ship:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não encontrei duas pessoas disponíveis para o Ship.",
			)

			return true
		}

	// ======================================================
	// UMA MARCAÇÃO
	//
	// Autor + pessoa marcada.
	// ======================================================

	case 1:
		first =
			shipPlayer{
				JID: msgEvent.Info.
					Sender.
					ToNonAD(),

				Name: strings.TrimSpace(
					msgEvent.Info.PushName,
				),
			}

		if first.Name == "" {
			first.Name =
				first.JID.User
		}

		second, err =
			resolveShipPlayer(
				client,
				groupInfo.Participants,
				mentioned[0],
			)

		if err != nil {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				err.Error(),
			)

			return true
		}

	// ======================================================
	// DUAS MARCAÇÕES
	//
	// Pessoa 1 + Pessoa 2.
	// ======================================================

	case 2:
		first, err =
			resolveShipPlayer(
				client,
				groupInfo.Participants,
				mentioned[0],
			)

		if err != nil {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				err.Error(),
			)

			return true
		}

		second, err =
			resolveShipPlayer(
				client,
				groupInfo.Participants,
				mentioned[1],
			)

		if err != nil {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				err.Error(),
			)

			return true
		}

	default:
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ O *!ship* aceita no máximo duas pessoas.",
		)

		return true
	}

	if funSameJID(
		first.JID,
		second.JID,
	) {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"😂 Não dá para fazer Ship de uma pessoa com ela mesma.",
		)

		return true
	}

	percentage :=
		shipCompatibility(
			first.JID,
			second.JID,
		)

	response := fmt.Sprintf(
		"💘 *SHIP* 💘\n\n"+
			"@%s ❤️ @%s\n\n"+
			"*Compatibilidade:* %d%%\n"+
			"%s\n\n"+
			"%s",
		first.Name,
		second.Name,
		percentage,
		shipProgressBar(
			percentage,
		),
		shipCompatibilityMessage(
			percentage,
		),
	)

	err =
		whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			response,
			[]types.JID{
				first.JID.ToNonAD(),
				second.JID.ToNonAD(),
			},
		)

	if err != nil {
		logger.Error(
			"Erro ao enviar resultado do !ship:",
			err,
		)

		return true
	}

	logger.Info(
		"Ship calculado:",
		first.JID.String(),
		"x",
		second.JID.String(),
		"Compatibilidade:",
		percentage,
	)

	return true
}

func shipMentionedJIDs(
	msgEvent *events.Message,
) []string {
	if msgEvent == nil ||
		msgEvent.Message == nil ||
		msgEvent.Message.ExtendedTextMessage == nil ||
		msgEvent.Message.
			ExtendedTextMessage.
			ContextInfo == nil {

		return nil
	}

	return msgEvent.Message.
		ExtendedTextMessage.
		ContextInfo.
		GetMentionedJID()
}

func resolveShipPlayer(
	client *whatsmeow.Client,
	participants []types.GroupParticipant,
	rawJID string,
) (
	shipPlayer,
	error,
) {
	resolvedJID, ok :=
		resolveParticipantJID(
			participants,
			rawJID,
		)

	if !ok {
		return shipPlayer{},
			fmt.Errorf(
				"❌ Uma das pessoas mencionadas não pertence a este grupo.",
			)
	}

	jid, err :=
		types.ParseJID(
			resolvedJID,
		)

	if err != nil {
		return shipPlayer{},
			fmt.Errorf(
				"❌ Não foi possível identificar uma das pessoas mencionadas.",
			)
	}

	participant, ok :=
		findFunParticipant(
			participants,
			jid,
		)

	if !ok {
		return shipPlayer{},
			fmt.Errorf(
				"❌ Não foi possível identificar uma das pessoas mencionadas.",
			)
	}

	if funIsBotParticipant(
		client,
		participant,
	) {
		return shipPlayer{},
			fmt.Errorf(
				"🤖 O bot não participa do Ship. Escolha outra pessoa.",
			)
	}

	name :=
		strings.TrimSpace(
			funParticipantDisplayName(
				participant,
			),
		)

	if name == "" {
		name = jid.User
	}

	return shipPlayer{
		JID:  jid.ToNonAD(),
		Name: name,
	}, nil
}

func randomShipPair(
	client *whatsmeow.Client,
	participants []types.GroupParticipant,
) (
	shipPlayer,
	shipPlayer,
	error,
) {
	candidates :=
		make(
			[]shipPlayer,
			0,
			len(participants),
		)

	seen :=
		make(
			map[string]struct{},
		)

	for _, participant := range participants {

		if funIsBotParticipant(
			client,
			participant,
		) {
			continue
		}

		jid :=
			shipParticipantJID(
				participant,
			)

		if jid.User == "" {
			continue
		}

		key :=
			jid.
				ToNonAD().
				String()

		if _, exists :=
			seen[key]; exists {

			continue
		}

		seen[key] =
			struct{}{}

		name :=
			strings.TrimSpace(
				funParticipantDisplayName(
					participant,
				),
			)

		if name == "" {
			name = jid.User
		}

		candidates =
			append(
				candidates,
				shipPlayer{
					JID:  jid.ToNonAD(),
					Name: name,
				},
			)
	}

	if len(candidates) < 2 {
		return shipPlayer{},
			shipPlayer{},
			fmt.Errorf(
				"menos de dois participantes disponíveis",
			)
	}

	firstIndex, err :=
		shipRandomIndex(
			len(candidates),
		)

	if err != nil {
		return shipPlayer{},
			shipPlayer{},
			err
	}

	first :=
		candidates[firstIndex]

	candidates =
		append(
			candidates[:firstIndex],
			candidates[firstIndex+1:]...,
		)

	secondIndex, err :=
		shipRandomIndex(
			len(candidates),
		)

	if err != nil {
		return shipPlayer{},
			shipPlayer{},
			err
	}

	return first,
		candidates[secondIndex],
		nil
}

func shipParticipantJID(
	participant types.GroupParticipant,
) types.JID {
	if participant.PhoneNumber.User != "" {
		return participant.
			PhoneNumber.
			ToNonAD()
	}

	if participant.JID.User != "" {
		return participant.
			JID.
			ToNonAD()
	}

	return participant.
		LID.
		ToNonAD()
}

func shipRandomIndex(
	length int,
) (
	int,
	error,
) {
	if length <= 0 {
		return 0,
			fmt.Errorf(
				"lista vazia",
			)
	}

	n, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(
				int64(length),
			),
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro ao realizar sorteio: %w",
				err,
			)
	}

	return int(
			n.Int64(),
		),
		nil
}

// shipCompatibility gera uma porcentagem fixa para a dupla.
//
// A ordem dos participantes não altera o resultado.
//
// Exemplo:
//
// Ana + João = 83%
// João + Ana = 83%
func shipCompatibility(
	first types.JID,
	second types.JID,
) int {
	pair := []string{
		first.
			ToNonAD().
			String(),

		second.
			ToNonAD().
			String(),
	}

	sort.Strings(pair)

	sum :=
		sha256.Sum256(
			[]byte(
				pair[0] +
					"|" +
					pair[1],
			),
		)

	value :=
		binary.BigEndian.
			Uint64(
				sum[:8],
			)

	return int(
		value % 101,
	)
}

func shipProgressBar(
	percentage int,
) string {
	if percentage < 0 {
		percentage = 0
	}

	if percentage > 100 {
		percentage = 100
	}

	filled :=
		percentage / 10

	if percentage > 0 &&
		filled == 0 {

		filled = 1
	}

	if percentage == 100 {
		filled = 10
	}

	return strings.Repeat(
		"█",
		filled,
	) +
		strings.Repeat(
			"░",
			10-filled,
		)
}

func shipCompatibilityMessage(
	percentage int,
) string {
	switch {

	case percentage <= 10:
		return "💀 Melhor deixar só na amizade..."

	case percentage <= 30:
		return "😬 Essa combinação vai exigir bastante esforço."

	case percentage <= 50:
		return "🤔 Talvez role alguma coisa por aqui..."

	case percentage <= 70:
		return "💕 Tem potencial!"

	case percentage <= 90:
		return "❤️ Casalzão! A química apareceu."

	case percentage < 100:
		return "🔥 Isso aqui está perigoso!"

	default:
		return "💍 100%! Pode marcar o casamento!"
	}
}
