package auth

import (
	"encoding/json"
	"os"
	"strings"

	"whatsapp-sticker-bot/internal/logger"
)

type GroupConfig struct {
	AllowedGroups []string `json:"allowed_groups"`
}

var allowedGroups = map[string]bool{}

func normalizeJID(jid string) string {
	return strings.TrimSpace(strings.ToLower(jid))
}

func LoadGroups(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg GroupConfig

	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	for _, group := range cfg.AllowedGroups {
		allowedGroups[normalizeJID(group)] = true
	}

	logger.Info("Whitelist carregada:", len(allowedGroups), "grupo(s) autorizado(s)")

	return nil
}

func IsGroupAllowed(groupID string) bool {
	return allowedGroups[normalizeJID(groupID)]
}
