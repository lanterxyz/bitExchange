package server_test

import (
	"testing"

	"bitExchange/internal/server"
)

func TestRelayManagerCreatesAndRemovesSession(t *testing.T) {
	mgr := server.NewRelayManager(server.RelayConfig{
		Enabled:     true,
		MaxBytes:    1024,
		MaxSessions: 10,
	})

	sessionID, err := mgr.CreateSession("device-a", "device-b")
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if sessionID == "" {
		t.Fatalf("sessionID should not be empty")
	}
	if mgr.ActiveSessionCount() != 1 {
		t.Fatalf("ActiveSessionCount = %d, want 1", mgr.ActiveSessionCount())
	}

	_, ok := mgr.GetSession(sessionID)
	if !ok {
		t.Fatalf("GetSession should find the session")
	}

	mgr.RemoveSession(sessionID)
	if mgr.ActiveSessionCount() != 0 {
		t.Fatalf("ActiveSessionCount = %d after remove, want 0", mgr.ActiveSessionCount())
	}
}

func TestRelayManagerRejectsWhenDisabled(t *testing.T) {
	mgr := server.NewRelayManager(server.RelayConfig{Enabled: false})
	_, err := mgr.CreateSession("device-a", "device-b")
	if err == nil {
		t.Fatalf("CreateSession should have failed when relay is disabled")
	}
}

func TestRelayManagerRejectsWhenFull(t *testing.T) {
	mgr := server.NewRelayManager(server.RelayConfig{
		Enabled:     true,
		MaxSessions: 2,
	})

	mgr.CreateSession("a", "b")
	mgr.CreateSession("c", "d")
	_, err := mgr.CreateSession("e", "f")
	if err == nil {
		t.Fatalf("CreateSession should have failed when at max sessions")
	}
}
