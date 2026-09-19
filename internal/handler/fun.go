package handler

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var kissMessages = []string{
	"💋 @%s deu um beijo em @%s! O clima ficou romântico por aqui. 😳❤️",
	"😘 @%s não resistiu e mandou um beijão para @%s!",
	"💘 @%s se aproximou de @%s e... *SMACK!* 💋",
	"🥰 @%s acabou de dar um beijo carinhoso em @%s!",
	"😏 @%s olhou para @%s, chegou mais perto e roubou um beijo! 💋",
	"❤️ @%s mandou um beijo tão caprichado para @%s que até o grupo sentiu!",
	"💞 @%s e @%s protagonizaram uma cena digna de novela: teve beijo! 📺💋",
	"🌹 @%s chegou com todo o charme e deu um beijo em @%s!",
	"🔥 @%s deu um beijo em @%s e a temperatura do grupo subiu alguns graus!",
	"😳 @%s beijou @%s do nada! Alguém chama os roteiristas dessa história! 😂",
	"💌 @%s tinha uma mensagem especial para @%s... mas resolveu entregar com um beijo! 💋",
	"🫣 @%s deu aquele beijo surpresa em @%s! Ninguém estava preparado para isso.",
	"✨ @%s deu um beijo em @%s. Foi tão bonito que quase apareceu uma trilha sonora.",
	"😚 @%s mandou um beijo direto para @%s. Entrega realizada com sucesso! ✅",
	"🎯 @%s mirou em @%s e acertou em cheio... com um beijo! 💋❤️",
}

var slapMessages = []string{
	"👋 @%s deu um tapa cinematográfico em @%s! Até o grupo ficou em silêncio. 😂",
	"💥 @%s acertou um tapa em @%s! O barulho ecoou pelo WhatsApp inteiro.",
	"😂 @%s olhou para @%s e decidiu resolver a situação com um belo *TAPA!*",
	"🖐️ @%s aplicou o famoso tapa educativo em @%s! 📚",
	"😵 @%s deu um tapa em @%s e a alma quase saiu para dar uma volta.",
	"🎬 @%s acertou @%s com um tapa digno de cena de novela! 💥",
	"🚨 Atenção! @%s acabou de distribuir um tapa gratuito para @%s! 😂",
	"⚡ @%s foi mais rápido que o Wi-Fi e acertou um tapa em @%s!",
	"🥴 @%s deu um tapa em @%s. Dizem que até agora está procurando de onde veio.",
	"🏓 @%s acertou um tapa em @%s com tanta precisão que parecia competição profissional.",
	"👀 @%s encarou @%s por alguns segundos... e então veio o tapa! 👋",
	"💫 @%s deu um tapa em @%s e apareceram até estrelinhas igual desenho animado.",
	"📢 *PLAFT!* @%s acabou de acertar um tapa em @%s!",
	"🤣 @%s resolveu reiniciar @%s manualmente: um tapa foi aplicado com sucesso.",
	"🔧 @%s tentou corrigir o problema em @%s utilizando a tradicional manutenção percussiva: *TAPA!* 😂",
}

var hugMessages = []string{
	"🤗 @%s deu aquele abraço apertado em @%s! ❤️",
	"🫂 @%s puxou @%s para um abraço daqueles que recarregam a bateria.",
	"🥰 @%s decidiu que @%s precisava de um abraço imediatamente!",
	"💞 @%s abraçou @%s e o nível de fofura do grupo aumentou.",
	"✨ @%s deu um abraço em @%s. Momento wholesome desbloqueado!",
	"🐻 @%s aplicou um abraço de urso em @%s!",
	"❤️ @%s abriu os braços e @%s ganhou um abraço grátis!",
	"🌈 @%s mandou energias boas para @%s em forma de abraço.",
	"😌 @%s abraçou @%s. Por alguns segundos, tudo ficou em paz.",
	"🎁 @%s entregou para @%s o melhor presente possível: um abraço!",
	"🫶 @%s chegou junto e deu um abraço carinhoso em @%s.",
	"😊 @%s encontrou @%s pelo grupo e resolveu distribuir carinho.",
	"💖 @%s abraçou @%s com 100% de sinceridade e 0% de cooldown.",
	"🔋 @%s deu um abraço em @%s e restaurou +100 de energia emocional.",
	"☁️ @%s deu um abraço tão confortável em @%s que parecia travesseiro.",
}

var biteMessages = []string{
	"🦷 @%s deu uma mordidinha em @%s! Ninguém sabe o motivo. 😂",
	"😈 @%s olhou para @%s e decidiu: hoje vai ter mordida!",
	"🧛 @%s ativou o modo vampiro e mordeu @%s!",
	"😂 @%s simplesmente mordeu @%s e se recusou a explicar.",
	"🐊 @%s confundiu o grupo com um documentário e mordeu @%s!",
	"🤨 @%s deu uma mordida em @%s. Comportamento absolutamente normal.",
	"🚨 Atenção: @%s acaba de atacar @%s com uma mordidinha surpresa!",
	"😳 @%s chegou perto de @%s e... *NHAC!*",
	"🐹 @%s deu uma mordidinha em @%s digna de hamster bravo.",
	"🍪 @%s aparentemente confundiu @%s com um biscoito e deu uma mordida.",
	"🤣 @%s mordeu @%s. O RH do grupo já foi notificado.",
	"🦈 @%s apareceu do nada e deu uma mordida em @%s!",
	"👀 @%s encarou @%s por alguns segundos antes do inevitável *NHAC!*",
	"🎯 @%s encontrou o alvo perfeito para uma mordidinha: @%s!",
	"⚠️ @%s mordeu @%s. Recomenda-se manter distância de segurança.",
}

type funParticipant struct {
	JID         types.JID
	DisplayName string
}

func handleFunCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts := strings.Fields(text)

	if len(parts) == 0 {
		return false
	}

	command := strings.ToLower(parts[0])

	if command != "!beijo" &&
		command != "!tapa" &&
		command != "!abraco" &&
		command != "!morder" {
		return false
	}

	// ======================================================
	// APENAS GRUPOS
	// ======================================================

	if !msgEvent.Info.IsGroup {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Este comando só pode ser utilizado em grupos.",
		)

		return true
	}

	mentionedJIDs := funMentionedJIDs(msgEvent)

	// ======================================================
	// VALIDAÇÃO DE MENÇÕES
	// ======================================================

	if len(mentionedJIDs) > 1 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			funUsageMessage(command),
		)

		return true
	}

	// Se não houver menção, o comando precisa estar sozinho.
	//
	// Válido:
	//
	// !beijo
	//
	// Inválido:
	//
	// !beijo fulano
	if len(mentionedJIDs) == 0 &&
		len(parts) != 1 {

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			funUsageMessage(command),
		)

		return true
	}

	// ======================================================
	// PARTICIPANTES DO GRUPO
	// ======================================================

	groupInfo, err := client.GetGroupInfo(
		context.Background(),
		msgEvent.Info.Chat,
	)

	if err != nil {
		logger.Error(
			"Erro ao buscar participantes do grupo:",
			err,
		)

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Não foi possível consultar os participantes do grupo.",
		)

		return true
	}

	// ======================================================
	// AUTOR DO COMANDO
	// ======================================================

	senderJID := msgEvent.Info.Sender.ToNonAD()

	senderName := strings.TrimSpace(
		msgEvent.Info.PushName,
	)

	if senderName == "" {
		senderName = funJIDDisplayName(
			senderJID,
		)
	}

	// ======================================================
	// DEFINIR ALVO
	// ======================================================

	var target funParticipant

	// ------------------------------------------------------
	// COM MENÇÃO
	// ------------------------------------------------------

	if len(mentionedJIDs) == 1 {
		mentionedJID, err := types.ParseJID(
			mentionedJIDs[0],
		)

		if err != nil {
			logger.Error(
				"Erro ao interpretar participante mencionado:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não foi possível identificar a pessoa mencionada.",
			)

			return true
		}

		mentionedJID = mentionedJID.ToNonAD()

		resolved, found := findFunParticipant(
			groupInfo.Participants,
			mentionedJID,
		)

		if !found {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ A pessoa mencionada não pertence a este grupo.",
			)

			return true
		}

		// Não permitir beijo/tapa em si mesmo.
		if funIsSenderParticipant(
			msgEvent,
			resolved,
		) {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"😅 Você precisa escolher outra pessoa.",
			)

			return true
		}

		target = funParticipant{
			JID: resolved.JID.ToNonAD(),

			DisplayName: funMentionLabel(
				text,
				command,
			),
		}

		if target.DisplayName == "" {
			target.DisplayName = funParticipantDisplayName(
				resolved,
			)
		}
	} else {

		// --------------------------------------------------
		// SEM MENÇÃO
		// --------------------------------------------------
		//
		// Sorteia qualquer participante do grupo,
		// exceto:
		//
		// - quem executou o comando;
		// - o próprio bot.
		//
		// --------------------------------------------------

		target, err = randomFunParticipant(
			client,
			msgEvent,
			groupInfo.Participants,
		)

		if err != nil {
			logger.Error(
				"Erro ao sortear participante:",
				err,
			)

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Não encontrei outra pessoa disponível no grupo.",
			)

			return true
		}
	}

	// ======================================================
	// SELECIONAR VARIAÇÃO
	// ======================================================

	var templates []string

	switch command {
	case "!beijo":
		templates = kissMessages

	case "!tapa":
		templates = slapMessages

	case "!abraco":
		templates = hugMessages

	case "!morder":
		templates = biteMessages
	}

	template := randomFunMessage(
		templates,
	)

	response := fmt.Sprintf(
		template,
		senderName,
		target.DisplayName,
	)

	// ======================================================
	// ENVIAR
	// ======================================================

	err = whatsapp.SendMentionedText(
		client,
		msgEvent.Info.Chat,
		response,
		[]types.JID{
			senderJID,
			target.JID,
		},
	)

	if err != nil {
		logger.Error(
			"Erro ao enviar comando recreativo:",
			err,
		)

		return true
	}

	logger.Info(
		"Comando recreativo executado:",
		command,
		"por:",
		senderJID.String(),
		"alvo:",
		target.JID.String(),
	)

	return true
}

// findFunParticipant procura um participante considerando:
//
// - JID;
// - PhoneNumber;
// - LID.
//
// Isso é importante porque o WhatsApp pode utilizar
// identificadores diferentes dependendo da mensagem.
func findFunParticipant(
	participants []types.GroupParticipant,
	jid types.JID,
) (types.GroupParticipant, bool) {
	for _, participant := range participants {
		if funParticipantMatchesJID(
			participant,
			jid,
		) {
			return participant, true
		}
	}

	return types.GroupParticipant{}, false
}

// randomFunParticipant sorteia uma pessoa do grupo,
// removendo o autor do comando e o próprio bot.
func randomFunParticipant(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	participants []types.GroupParticipant,
) (funParticipant, error) {
	candidates := make(
		[]funParticipant,
		0,
		len(participants),
	)

	seen := make(
		map[string]struct{},
	)

	for _, participant := range participants {
		jid := participant.JID.ToNonAD()

		if jid.User == "" {
			continue
		}

		// Não selecionar o próprio usuário.
		if funIsSenderParticipant(
			msgEvent,
			participant,
		) {
			continue
		}

		// Não selecionar o bot.
		if funIsBotParticipant(
			client,
			participant,
		) {
			continue
		}

		key := jid.String()

		// Proteção contra participantes duplicados.
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}

		candidates = append(
			candidates,
			funParticipant{
				JID: jid,

				DisplayName: funParticipantDisplayName(
					participant,
				),
			},
		)
	}

	if len(candidates) == 0 {
		return funParticipant{},
			fmt.Errorf(
				"nenhum participante disponível para sorteio",
			)
	}

	n, err := rand.Int(
		rand.Reader,
		big.NewInt(
			int64(len(candidates)),
		),
	)

	if err != nil {
		return funParticipant{},
			fmt.Errorf(
				"erro ao sortear participante: %w",
				err,
			)
	}

	return candidates[int(n.Int64())], nil
}

// funParticipantMatchesJID verifica os possíveis
// identificadores associados ao participante.
func funParticipantMatchesJID(
	participant types.GroupParticipant,
	jid types.JID,
) bool {
	return funSameJID(
		participant.JID,
		jid,
	) ||
		funSameJID(
			participant.PhoneNumber,
			jid,
		) ||
		funSameJID(
			participant.LID,
			jid,
		)
}

// funIsSenderParticipant impede que quem executou
// o comando seja selecionado como alvo.
func funIsSenderParticipant(
	msgEvent *events.Message,
	participant types.GroupParticipant,
) bool {
	if funParticipantMatchesJID(
		participant,
		msgEvent.Info.Sender,
	) {
		return true
	}

	if msgEvent.Info.SenderAlt.User != "" &&
		funParticipantMatchesJID(
			participant,
			msgEvent.Info.SenderAlt,
		) {

		return true
	}

	return false
}

// funIsBotParticipant impede que a própria conta
// conectada ao WhatsApp seja sorteada.
func funIsBotParticipant(
	client *whatsmeow.Client,
	participant types.GroupParticipant,
) bool {
	if client == nil ||
		client.Store == nil {

		return false
	}

	if client.Store.ID != nil &&
		funParticipantMatchesJID(
			participant,
			*client.Store.ID,
		) {

		return true
	}

	if client.Store.LID.User != "" &&
		funParticipantMatchesJID(
			participant,
			client.Store.LID,
		) {

		return true
	}

	return false
}

// funParticipantDisplayName escolhe o texto exibido
// quando a pessoa foi sorteada aleatoriamente.
//
// Preferimos o número de telefone ao LID porque
// é muito mais legível na mensagem.
func funParticipantDisplayName(
	participant types.GroupParticipant,
) string {
	phone := participant.PhoneNumber.ToNonAD()

	if phone.User != "" {
		return phone.User
	}

	jid := participant.JID.ToNonAD()

	if jid.User != "" {
		return jid.User
	}

	lid := participant.LID.ToNonAD()

	if lid.User != "" {
		return lid.User
	}

	return "usuário"
}

// funMentionedJIDs retorna os JIDs mencionados
// explicitamente na mensagem.
func funMentionedJIDs(
	msgEvent *events.Message,
) []string {
	if msgEvent == nil ||
		msgEvent.Message == nil {

		return nil
	}

	extended := msgEvent.Message.GetExtendedTextMessage()

	if extended == nil {
		return nil
	}

	contextInfo := extended.GetContextInfo()

	if contextInfo == nil {
		return nil
	}

	return contextInfo.GetMentionedJID()
}

// funUsageMessage retorna a sintaxe correta
// de cada comando.
func funUsageMessage(
	command string,
) string {
	switch command {
	case "!beijo":
		return "💋 Use *!beijo @pessoa* ou apenas *!beijo* para escolher alguém aleatoriamente."

	case "!tapa":
		return "👋 Use *!tapa @pessoa* ou apenas *!tapa* para escolher alguém aleatoriamente."

	case "!abraco":
		return "🤗 Use *!abraco @pessoa* ou apenas *!abraco* para escolher alguém aleatoriamente."

	case "!morder":
		return "🦷 Use *!morder @pessoa* ou apenas *!morder* para escolher alguém aleatoriamente."

	default:
		return "❌ Comando inválido."
	}
}

// funMentionLabel aproveita o nome digitado/exibido
// na própria menção.
//
// Exemplo:
//
// !beijo @João
//
// retorna:
//
// João
func funMentionLabel(
	text string,
	command string,
) string {
	text = strings.TrimSpace(
		text,
	)

	if len(text) <= len(command) {
		return ""
	}

	remaining := strings.TrimSpace(
		text[len(command):],
	)

	remaining = strings.TrimPrefix(
		remaining,
		"@",
	)

	return strings.TrimSpace(
		remaining,
	)
}

// funJIDDisplayName fornece um fallback caso
// PushName esteja vazio.
func funJIDDisplayName(
	jid types.JID,
) string {
	jid = jid.ToNonAD()

	name := strings.TrimSpace(
		jid.User,
	)

	if name == "" {
		return "usuário"
	}

	return name
}

// funSameJID compara dois JIDs removendo
// identificadores de dispositivo.
func funSameJID(
	a types.JID,
	b types.JID,
) bool {
	if a.User == "" ||
		b.User == "" {

		return false
	}

	return a.ToNonAD().String() ==
		b.ToNonAD().String()
}

// randomFunMessage seleciona aleatoriamente
// uma das 15 variações do comando.
func randomFunMessage(
	messages []string,
) string {
	if len(messages) == 0 {
		return "@%s interagiu com @%s!"
	}

	if len(messages) == 1 {
		return messages[0]
	}

	n, err := rand.Int(
		rand.Reader,
		big.NewInt(
			int64(len(messages)),
		),
	)

	if err != nil {
		logger.Warn(
			"Erro ao sortear mensagem recreativa:",
			err,
		)

		return messages[0]
	}

	return messages[int(n.Int64())]
}
