package handler

import (
	"strings"

	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

const menuMessage = `⚔️ *GRIMÓRIO DO AVENTUREIRO*
🏰 *v3.5.0*

👑 *AVENTUREIRO*
👤 !perfil
↳ !ficha • !personagem
🏆 !conquistas
✅ !conquistas finalizadas
🔒 !conquistas bloqueadas
💰 !ranking

💰 *ECONOMIA*
🪙 !gold • 💰 !saldo
🤝 !pix @pessoa <valor>
🏦 !cofre
↳ guardar/depositar/colocar
↳ retirar/sacar/pegar
🦹 !roubar @pessoa
🛡️ !escudo
🌪️ !pressagio

🎲 *JOGOS*
🎲 !bet <valor>
🎰 !slots <valor>
🪙 !caraoucoroa <valor> <cara|coroa>
⚔️ !duelo @pessoa <valor>
🍀 !sorte
🎟️ !loteria <quantidade>
📚 !quiz
🎯 !forca

💡 Valores flexíveis nas apostas:
20% • metade • all • tudo • 1.000.000

🗺️ *RPG*
⚔️ !rpg

🎒 !inventario [filtro] [página]
↳ !inv • !mochila • !bolsa
↳ equipamentos • materiais • equipado
↳ arma • escudo • armadura
↳ comum • raro • épico • lendário...

🛡️ !equipamentos
↳ !arsenal • !gear

🐺 !pve [região]
↳ !caçar • !lutar • !combater

🗺️ Regiões:
floresta/mata/bosque
pedreira/rocha
mina/mineração

🏚️ !dungeons
↳ !dungeon • !masmorra
↳ !entrar <dungeon>

💎 !cristais

🛒 *EQUIPAMENTOS*
🏪 !loja
🛒 !comprar <código>
↳ !adquirir <código>

⚒️ !forja
🔥 !forjar <código>
↳ !fabricar • !criar

⚔️ !equipar <código>
↳ !vestir <código>

📦 !desequipar <slot>
↳ !tirar • !desvestir
Slots: arma • escudo • armadura

🍻 *DIVERSÃO*
💘 !ship
💋 !beijo @pessoa
🤗 !abraco @pessoa
👋 !tapa @pessoa
🦷 !morder @pessoa
🎨 !f

━━━━━━━━━━━━━━━━━━━━
📖 Use *!menu* sempre que precisar.
💡 Os aliases são opcionais: os comandos
originais continuam funcionando normalmente.`

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
