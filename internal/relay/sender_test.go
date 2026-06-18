package relay_test

import (
	"testing"

	"bitExchange/internal/relay"
)

func TestRelaySendRejectsExceedingMaxBytes(t *testing.T) {
	sender := relay.NewSender(relay.SenderConfig{RelayMaxBytes: 100})

	largePayload := make([]byte, 200)
	err := sender.SendText("http://example.com", "relay-session-1", "device-b", string(largePayload))
	if err == nil {
		t.Fatalf("SendText should have rejected payload exceeding max bytes")
	}
}

func TestRelaySendAcceptsTextWithinLimit(t *testing.T) {
	sender := relay.NewSender(relay.SenderConfig{RelayMaxBytes: 1024})

	// Will fail at connection level but NOT at size check level
	err := sender.SendText("http://127.0.0.1:1", "relay-session-1", "device-b", "hello")
	if err == nil {
		t.Fatalf("expected connection error on invalid URL")
	}
	// Verify the error is a connection error, not a size rejection
	if err.Error()[:13] == "relay refused" {
		t.Fatalf("should be connection error, got size rejection: %v", err)
	}
}
