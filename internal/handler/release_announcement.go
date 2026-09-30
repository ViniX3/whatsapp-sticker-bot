package handler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"whatsapp-sticker-bot/internal/auth"
	"whatsapp-sticker-bot/internal/database"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

const currentReleaseAnnouncementID = "2026-09-30-rpg-rebalance-v1"

const currentReleaseAnnouncementMessage = `🤖 *BOT ATUALIZADO!*

⚔️ *GRANDE REBALANCEAMENTO DO RPG*

📈 *Novo sistema de Poder de Combate (PC)*
• O PC de todos os jogadores foi recalculado para uma nova escala.
• O *nível do personagem* agora possui uma participação muito maior no seu poder.
• Armas, escudos e armaduras foram rebalanceados.
• Equipamentos de raridades maiores agora possuem diferenças de poder mais significativas.
• Seu PC poderá estar *bem diferente do valor anterior*. Isso é esperado.
• *Nenhum equipamento ou progresso foi perdido.*

🏰 *Dungeons*
• As Dungeons foram rebalanceadas para acompanhar a nova progressão de PC.
• Quanto mais avançada a Dungeon, maior será o poder necessário para enfrentá-la.

👹 *PvE*
• Os inimigos também foram adaptados à nova escala.
• A progressão agora acompanha melhor a evolução dos jogadores e equipamentos.

🐉 *Raid Boss*
• O sistema de Raid recebeu uma grande expansão.
• Bosses *Legendary, Mythic e Sacred* fazem parte da progressão.
• O poder dos Bosses se adapta aos jogadores mais fortes do grupo.
• Até *10 jogadores* podem lutar juntos.
• Raids possuem recompensas próprias e rotação automática de Bosses.

⚒️ *Progressão*
A progressão de equipamentos agora segue uma escala mais definida:

*PvE → Forja → Dungeons → Mercado Arcano → Raid*

✨ Esta nova escala também prepara o RPG para conteúdos ainda maiores e para a futura progressão *pós-Presságio*.

Use *!perfil* para conferir seu novo Poder de Combate.

⚔️ *Uma nova escala de poder começou.*`

func StartReleaseAnnouncement(
	client *whatsmeow.Client,
) {
	if client == nil {
		return
	}

	go announceCurrentRelease(
		client,
	)
}

func announceCurrentRelease(
	client *whatsmeow.Client,
) {
	if err :=
		ensureReleaseAnnouncementSchema(); err != nil {

		logger.Error(
			"Erro inicializando anúncios de atualização:",
			err,
		)

		return
	}

	groups :=
		auth.AllowedGroupIDs()

	if len(groups) == 0 {
		logger.Info(
			"Nenhum grupo autorizado para anúncio de atualização",
		)

		return
	}

	logger.Info(
		"Verificando anúncio da versão:",
		currentReleaseAnnouncementID,
		"Grupos:",
		len(groups),
	)

	for _, groupJID := range groups {
		announced, err :=
			releaseAlreadyAnnounced(
				currentReleaseAnnouncementID,
				groupJID,
			)

		if err != nil {
			logger.Error(
				"Erro verificando anúncio de atualização:",
				groupJID,
				err,
			)

			continue
		}

		if announced {
			logger.Debug(
				"Atualização já anunciada:",
				currentReleaseAnnouncementID,
				groupJID,
			)

			continue
		}

		jid, err :=
			types.ParseJID(
				groupJID,
			)

		if err != nil {
			logger.Error(
				"JID inválido no anúncio de atualização:",
				groupJID,
				err,
			)

			continue
		}

		// Confirma que o bot ainda possui acesso ao grupo antes
		// de tentar enviar a mensagem de atualização.
		if _, err :=
			client.GetGroupInfo(
				context.Background(),
				jid,
			); err != nil {

			logger.Warn(
				"Grupo ignorado no anúncio de atualização; bot sem acesso:",
				groupJID,
				err,
			)

			continue
		}

		if err :=
			whatsapp.SendText(
				client,
				jid,
				currentReleaseAnnouncementMessage,
			); err != nil {

			logger.Error(
				"Erro enviando anúncio de atualização:",
				groupJID,
				err,
			)

			continue
		}

		if err :=
			markReleaseAnnounced(
				currentReleaseAnnouncementID,
				groupJID,
			); err != nil {

			logger.Error(
				"Anúncio enviado, mas não foi possível registrar no banco:",
				groupJID,
				err,
			)

			continue
		}

		logger.Success(
			"Atualização anunciada:",
			currentReleaseAnnouncementID,
			"Grupo:",
			groupJID,
		)

		time.Sleep(
			750 * time.Millisecond,
		)
	}
}

func ensureReleaseAnnouncementSchema() error {
	if database.DB == nil {
		return fmt.Errorf(
			"banco de dados não inicializado",
		)
	}

	_, err :=
		database.DB.Exec(`
			CREATE TABLE IF NOT EXISTS bot_release_announcements (
				release_id TEXT NOT NULL,
				group_jid TEXT NOT NULL,
				announced_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,

				PRIMARY KEY (
					release_id,
					group_jid
				)
			)
		`)

	if err != nil {
		return fmt.Errorf(
			"erro criando tabela de anúncios: %w",
			err,
		)
	}

	return nil
}

func releaseAlreadyAnnounced(
	releaseID string,
	groupJID string,
) (
	bool,
	error,
) {
	var marker int

	err :=
		database.DB.QueryRow(`
			SELECT 1
			FROM bot_release_announcements
			WHERE release_id = ?
			  AND group_jid = ?
			LIMIT 1
		`,
			releaseID,
			groupJID,
		).Scan(
			&marker,
		)

	if err == sql.ErrNoRows {
		return false,
			nil
	}

	if err != nil {
		return false,
			fmt.Errorf(
				"erro consultando anúncio: %w",
				err,
			)
	}

	return true,
		nil
}

func markReleaseAnnounced(
	releaseID string,
	groupJID string,
) error {
	_, err :=
		database.DB.Exec(`
			INSERT OR IGNORE INTO bot_release_announcements (
				release_id,
				group_jid
			)
			VALUES (?, ?)
		`,
			releaseID,
			groupJID,
		)

	if err != nil {
		return fmt.Errorf(
			"erro registrando anúncio: %w",
			err,
		)
	}

	return nil
}
