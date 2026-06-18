package server

import (
	"fmt"
	"sync"
	"time"
)

type RelayConfig struct {
	Enabled        bool
	MaxBytes       int64
	MaxSessions    int
	SessionTimeout time.Duration
}

type relaySession struct {
	SessionID  string
	FromDevice string
	ToDevice   string
	CreatedAt  time.Time
}

type RelayManager struct {
	cfg      RelayConfig
	mu       sync.Mutex
	sessions map[string]relaySession
}

func NewRelayManager(cfg RelayConfig) *RelayManager {
	if cfg.SessionTimeout == 0 {
		cfg.SessionTimeout = 5 * time.Minute
	}
	if cfg.MaxSessions == 0 {
		cfg.MaxSessions = 100
	}
	return &RelayManager{
		cfg:      cfg,
		sessions: map[string]relaySession{},
	}
}

func (m *RelayManager) CreateSession(from, to string) (string, error) {
	if !m.cfg.Enabled {
		return "", fmt.Errorf("relay disabled")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sessions) >= m.cfg.MaxSessions {
		return "", fmt.Errorf("max relay sessions reached")
	}
	sessionID := fmt.Sprintf("relay-%d", time.Now().UnixNano())
	m.sessions[sessionID] = relaySession{
		SessionID:  sessionID,
		FromDevice: from,
		ToDevice:   to,
		CreatedAt:  time.Now(),
	}
	return sessionID, nil
}

func (m *RelayManager) RemoveSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

func (m *RelayManager) ActiveSessionCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

func (m *RelayManager) GetSession(sessionID string) (relaySession, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	return s, ok
}

func (m *RelayManager) Config() RelayConfig {
	return m.cfg
}
