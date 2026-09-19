package handler

import (
	"strings"

	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

const menuMessage = `🤖 *MENU DO BOT*

🎨 *FIGURINHAS*

*!f*
Cria uma figurinha a partir de imagem, vídeo ou GIF.

Envie a mídia com *!f* na legenda ou responda uma mídia com *!f*.


💰 *ECONOMIA GOLD*

*!gold*
Cria sua carteira e recebe o Gold inicial.

*!saldo*
Mostra seu saldo atual de Gold.

*!bet <valor>*
Aposta uma quantidade de Gold.

Exemplo:
*!bet 500*

*!pix @pessoa <valor>*
Transfere Gold para outra pessoa.

Exemplo:
*!pix @pessoa 1000*

*!roubar @pessoa*
Tenta roubar Gold de outro jogador.

*!escudo*
Compra proteção temporária contra roubos.

*!sorte*
Participa do sorteio diário e pode ganhar Gold.

*!ranking*
Mostra os jogadores mais ricos do grupo.


🧠 *QUIZ*

*!quiz*
Inicia uma pergunta aleatória.

São *600 perguntas* distribuídas em 6 dificuldades:

🟢 Super Fácil
🔵 Fácil
🟡 Médio
🟠 Difícil
🔴 Super Difícil
💀 Insano

Responda usando:

*A*
*B*
*C*
*D*

Quanto maior a dificuldade, maior a recompensa em Gold.


🎉 *DIVERSÃO*

*!beijo @pessoa*
Dá um beijo na pessoa marcada. 💋

*!beijo*
Escolhe alguém aleatoriamente no grupo. 😏

*!tapa @pessoa*
Dá um tapa na pessoa marcada. 👋

*!tapa*
Escolhe uma vítima aleatória no grupo. 😂


📖 *AJUDA*

*!menu*
Exibe este menu.


━━━━━━━━━━━━━━━━━━

💡 *Dica*

Os comandos *!beijo* e *!tapa* podem ser usados com ou sem marcação.

No modo aleatório, o bot escolhe outra pessoa do grupo automaticamente.

Boa diversão! 🎮`

// handleMenuCommand processa o comando !menu.
//
// Retorna true quando a mensagem corresponde ao comando,
// permitindo que ProcessMessage interrompa o processamento.
func handleMenuCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts := strings.Fields(text)

	if len(parts) == 0 {
		return false
	}

	if !strings.EqualFold(
		parts[0],
		"!menu",
	) {
		return false
	}

	if len(parts) != 1 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"📖 Uso correto: *!menu*",
		)

		return true
	}

	err := whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		menuMessage,
	)

	if err != nil {
		logger.Error(
			"Erro ao enviar menu:",
			err,
		)

		return true
	}

	logger.Info(
		"Menu enviado para:",
		msgEvent.Info.Chat.String(),
	)

	return true
}
