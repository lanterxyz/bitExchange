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

func TestRelayFileExceedingLimitIsRejected(t *testing.T) {
	store := server.NewMemoryStore()
	table := server.NewOnlineTable()
	relayMgr := server.NewRelayManager(server.RelayConfig{
		Enabled:     true,
		MaxBytes:    100,
		MaxSessions: 10,
	})
	handler := server.NewSignalingServer(store, table, relayMgr)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	entryA := isig.OnlineEntry{DeviceID: "device-a", DeviceName: "a", Fingerprint: "fp-a"}
	entryB := isig.OnlineEntry{DeviceID: "device-b", DeviceName: "b", Fingerprint: "fp-b"}
	aBody, _ := json.Marshal(entryA)
	bBody, _ := json.Marshal(entryB)
	http.Post(ts.URL+"/signaling/online", "application/json", bytes.NewReader(aBody))
	http.Post(ts.URL+"/signaling/online", "application/json", bytes.NewReader(bBody))

	createReq, _ := json.Marshal(map[string]string{
		"from_device": "device-a",
		"to_device":   "device-b",
	})
	resp, _ := http.Post(ts.URL+"/relay/create", "application/json", bytes.NewReader(createReq))
	var createResp struct{ SessionID string `json:"session_id"` }
	json.NewDecoder(resp.Body).Decode(&createResp)
	resp.Body.Close()

	// Send file exceeding limit
	payload := map[string]interface{}{
		"kind":       "relay_file",
		"session_id": createResp.SessionID,
		"file_name":  "test.bin",
		"file_size":  200,
		"body":       "",
	}
	body, _ := json.Marshal(payload)
	resp2, err := http.Post(ts.URL+"/relay/send", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("relay send error: %v", err)
	}
	if resp2.StatusCode == http.StatusOK {
		t.Fatalf("relay should have rejected file exceeding 100 byte limit")
	}
}
