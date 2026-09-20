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

*!forca*
Inicia uma partida coletiva do jogo da Forca.

*!letra <letra>*
Tenta revelar uma letra da palavra.

Exemplo:
*!letra a*

*!palavra <resposta>*
Tenta descobrir a palavra completa.

Exemplo:
*!palavra inteligência artificial*

🏆 Quem concluir recebe Gold e XP.


*!loteria*
Mostra a rodada atual da Loteria do grupo.

*!loteria <quantidade>*
Compra bilhetes da rodada atual.

💰 Cada bilhete custa *500 Gold*.
🎟️ Máximo de *20 bilhetes por jogador*.
🏆 O sorteio acontece automaticamente ao vender os 100 bilhetes.

Exemplo:
*!loteria 5*


*!caraoucoroa <valor> <cara|coroa>*
Aposte Gold em um lançamento de moeda com chance real de 50/50.

Exemplo:
*!caraoucoroa 1000 cara*

*!slots <valor>*
Joga no caça-níquel utilizando Gold.

Exemplo:
*!slots 500*

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


🏆 *PROGRESSÃO*

*!perfil*
Mostra nível, XP, Gold e estatísticas do jogador.

*!conquistas*
Mostra suas conquistas e o progresso de cada uma.

*!conquistas @pessoa*
Consulta as conquistas de outro jogador.


⚔️ *DUELOS*

*!duelo @pessoa <valor>*
Desafia outro jogador para um duelo valendo Gold.

Exemplo:
*!duelo @pessoa 1000*

*!aceitar*
Aceita um duelo pendente.

*!recusar*
Recusa um duelo pendente.

O vencedor é escolhido automaticamente e recebe todo o pote.


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

*!ship*
Sorteia duas pessoas do grupo e calcula a compatibilidade. 💘

*!ship @pessoa*
Calcula a compatibilidade entre você e a pessoa marcada.

*!ship @pessoa1 @pessoa2*
Calcula a compatibilidade entre duas pessoas.

A porcentagem de cada dupla é sempre a mesma.

*!beijo @pessoa*
Dá um beijo na pessoa marcada. 💋

*!beijo*
Escolhe alguém aleatoriamente no grupo. 😏

*!tapa @pessoa*
Dá um tapa na pessoa marcada. 👋

*!tapa*
Escolhe uma vítima aleatória no grupo. 😂

*!abraco @pessoa*
Dá um abraço na pessoa marcada. 🤗

*!abraco*
Abraça alguém aleatoriamente no grupo. 🫂

*!morder @pessoa*
Dá uma mordidinha na pessoa marcada. 🦷

*!morder*
Escolhe alguém aleatoriamente para morder. 😈


📖 *AJUDA*

*!menu*
Exibe este menu.


━━━━━━━━━━━━━━━━━━

💡 *Dica*

Os comandos *!beijo*, *!tapa*, *!abraco* e *!morder* podem ser usados com ou sem marcação.

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
