package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitExchange/internal/server"
	isig "bitExchange/internal/signaling"
)

func TestRelayCreateAndSendText(t *testing.T) {
	store := server.NewMemoryStore()
	table := server.NewOnlineTable()
	relayMgr := server.NewRelayManager(server.RelayConfig{
		Enabled:     true,
		MaxBytes:    1024,
		MaxSessions: 10,
	})
	handler := server.NewSignalingServer(store, table, relayMgr)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Register two devices
	entryA := isig.OnlineEntry{DeviceID: "device-a", DeviceName: "a", Fingerprint: "fp-a"}
	entryB := isig.OnlineEntry{DeviceID: "device-b", DeviceName: "b", Fingerprint: "fp-b"}
	aBody, _ := json.Marshal(entryA)
	bBody, _ := json.Marshal(entryB)
	http.Post(ts.URL+"/signaling/online", "application/json", bytes.NewReader(aBody))
	http.Post(ts.URL+"/signaling/online", "application/json", bytes.NewReader(bBody))

	// Create relay session
	createReq, _ := json.Marshal(map[string]string{
		"from_device": "device-a",
		"to_device":   "device-b",
	})
	resp, err := http.Post(ts.URL+"/relay/create", "application/json", bytes.NewReader(createReq))
	if err != nil {
		t.Fatalf("relay create error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("relay create status = %d", resp.StatusCode)
	}

	var createResp struct{ SessionID string `json:"session_id"` }
	json.NewDecoder(resp.Body).Decode(&createResp)
	resp.Body.Close()

	// Send text via relay
	textPayload := map[string]interface{}{
		"kind":       "relay_text",
		"session_id": createResp.SessionID,
		"body":       "hello from relay",
	}
	body, _ := json.Marshal(textPayload)
	resp2, err := http.Post(ts.URL+"/relay/send", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("relay send error: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("relay send status = %d", resp2.StatusCode)
	}
}
