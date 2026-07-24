package album

import (
	"sync"

	"whatsapp-sticker-bot/internal/media"

	"go.mau.fi/whatsmeow/types"
)

type Manager struct {
	mu      sync.Mutex
	medias  map[string][]media.Media
}

func NewManager() *Manager {

	return &Manager{
		medias: make(map[string][]media.Media),
	}
}


func (m *Manager) Add(
	jid types.JID,
	item media.Media,
) {

	m.mu.Lock()
	defer m.mu.Unlock()

	key := jid.String()

	m.medias[key] = append(
		m.medias[key],
		item,
	)
}


func (m *Manager) Get(
	jid types.JID,
) []media.Media {

	m.mu.Lock()
	defer m.mu.Unlock()

	key := jid.String()

	items := m.medias[key]

	delete(
		m.medias,
		key,
	)

	return items
}


func (m *Manager) Count(
	jid types.JID,
) int {

	m.mu.Lock()
	defer m.mu.Unlock()

	return len(
		m.medias[jid.String()],
	)
}
