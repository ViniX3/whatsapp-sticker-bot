package handler

import (
	"os"
	"path/filepath"
	"strings"

	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var raidBossImageExtensions = []string{
	".jpg",
	".jpeg",
	".png",
	".webp",
}

func raidBossImagePath(
	imageKey string,
) (
	string,
	bool,
) {
	imageKey =
		strings.TrimSpace(
			imageKey,
		)

	if imageKey == "" {
		return "",
			false
	}

	for _, extension := range raidBossImageExtensions {

		path :=
			filepath.Join(
				"assets",
				"bosses",
				imageKey+
					extension,
			)

		info, err :=
			os.Stat(
				path,
			)

		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		return path,
			true
	}

	return "",
		false
}

func sendRaidBossImage(
	client *whatsmeow.Client,
	jid types.JID,
	lobby *rpg.RaidLobby,
	caption string,
) bool {
	if lobby == nil {
		return false
	}

	imagePath, exists :=
		raidBossImagePath(
			lobby.Boss.ImageKey,
		)

	if !exists {
		logger.Warn(
			"Imagem do Raid Boss não encontrada:",
			lobby.Boss.ID,
			"ImageKey:",
			lobby.Boss.ImageKey,
		)

		return false
	}

	err :=
		whatsapp.SendImageFileWithCaption(
			client,
			jid,
			imagePath,
			caption,
		)

	if err != nil {
		logger.Error(
			"Erro ao enviar imagem do Raid Boss:",
			lobby.Boss.ID,
			err,
		)

		return false
	}

	logger.Info(
		"Raid Boss enviado com imagem:",
		lobby.Boss.ID,
		imagePath,
	)

	return true
}

func sendRaidBossAnnouncement(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	lobby *rpg.RaidLobby,
	caption string,
) bool {
	if msgEvent == nil {
		return false
	}

	return sendRaidBossImage(
		client,
		msgEvent.Info.Chat,
		lobby,
		caption,
	)
}
