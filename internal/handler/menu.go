package handler

import (
	"strings"

	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

const menuMessage = `╔════════════════════╗
⚔️ *GRIMÓRIO DO AVENTUREIRO*
🏰 *VERSÃO 3.3.0*
╚════════════════════╝

📜 *Saudações, aventureiro!*
Aqui estão os caminhos disponíveis neste reino.

━━━━━━━━━━━━━━━━━━━━
👑 *CRÔNICAS DO HERÓI*
━━━━━━━━━━━━━━━━━━━━

👤 *!perfil*
Veja seu nível, XP, Gold e estatísticas.

🏆 *!conquistas*
Consulte suas conquistas e progresso.

🏆 *!conquistas @pessoa*
Veja as conquistas de outro aventureiro.

💰 *!ranking*
Conheça os mais ricos do reino.

━━━━━━━━━━━━━━━━━━━━
💰 *TESOURO DO REINO*
━━━━━━━━━━━━━━━━━━━━

🪙 *!gold*
Crie sua carteira e receba o Gold inicial.

💰 *!saldo*
Consulte suas riquezas.

🤝 *!pix @pessoa <valor>*
Transfira Gold para outro aventureiro.

━━━━━━━━━━━━━━━━━━━━
🎲 *JOGOS DA TAVERNA*
━━━━━━━━━━━━━━━━━━━━

🎲 *!bet <valor>*
Arrisque seu Gold em uma aposta.

🎰 *!slots <valor>*
Tente a sorte no caça-níquel.

🪙 *!caraoucoroa <valor> <cara|coroa>*
Desafie a sorte em um lançamento de moeda.

🍀 *!sorte*
Receba sua recompensa diária.

🎟️ *!loteria*
Veja a rodada atual da Loteria.

🎟️ *!loteria <quantidade>*
Compre bilhetes para disputar o Jackpot.

━━━━━━━━━━━━━━━━━━━━
⚔️ *ARENA DOS GUERREIROS*
━━━━━━━━━━━━━━━━━━━━

⚔️ *!duelo @pessoa <valor>*
Desafie outro guerreiro valendo Gold.

✅ *!aceitar*
Aceite um duelo pendente.

❌ *!recusar*
Recuse o desafio.

━━━━━━━━━━━━━━━━━━━━
🧠 *PROVAS DO SÁBIO*
━━━━━━━━━━━━━━━━━━━━

📚 *!quiz*
Enfrente uma das *1.000 perguntas* do reino.

🎯 *!forca*
Inicie uma partida coletiva de Forca.

🔤 *!letra <letra>*
Tente revelar uma letra.

📜 *!palavra <resposta>*
Arrisque a palavra completa.

━━━━━━━━━━━━━━━━━━━━
🦹 *SUBMUNDO DO REINO*
━━━━━━━━━━━━━━━━━━━━

🗡️ *!roubar @pessoa*
Tente roubar Gold de outro aventureiro.

🛡️ *!escudo*
Proteja suas riquezas contra ladrões.

━━━━━━━━━━━━━━━━━━━━
🍻 *TAVERNA DOS VIAJANTES*
━━━━━━━━━━━━━━━━━━━━

💘 *!ship*
Descubra a compatibilidade entre aventureiros.

💋 *!beijo @pessoa*
🤗 *!abraco @pessoa*
👋 *!tapa @pessoa*
🦷 *!morder @pessoa*

Os comandos também funcionam sem marcação,
escolhendo alguém aleatoriamente.

━━━━━━━━━━━━━━━━━━━━
🔨 *OFICINA DO ARTÍFICE*
━━━━━━━━━━━━━━━━━━━━

🎨 *!f*
Transforme imagem, vídeo ou GIF em figurinha.

━━━━━━━━━━━━━━━━━━━━
🏰 *PORTÕES DO REINO RPG*
━━━━━━━━━━━━━━━━━━━━

🔒 *O Reino ainda está sendo preparado...*

Em breve, aventureiros poderão acessar:

🏪 Mercado Real
🔮 Mercado Arcano
🎒 Inventário
⚔️ Equipamentos
🐺 Caçadas PvE
🏚️ Dungeons
🐉 Bosses e Raids

━━━━━━━━━━━━━━━━━━━━

⚜️ *Que a sorte acompanhe sua jornada.*

📖 Use *!menu* sempre que precisar
consultar este grimório.`

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
