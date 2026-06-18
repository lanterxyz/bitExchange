package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bitExchange/internal/server"
	isig "bitExchange/internal/signaling"
)

func TestOnlineTableRegisterAndLookup(t *testing.T) {
	table := server.NewOnlineTable()

	entry := isig.OnlineEntry{
		DeviceID:      "device-a",
		DeviceName:    "desktop-a",
		Fingerprint:   "abc123",
		ListenPort:    9001,
		PrivateAddrs:  []string{"10.0.0.5:9001", "192.168.1.10:9001"},
		RelayEnabled:  true,
		RelayMaxBytes: 67108864,
	}

	table.Register(entry)

	found, ok := table.Lookup("device-a")
	if !ok {
		t.Fatalf("Lookup did not find device-a")
	}
	if found.DeviceName != "desktop-a" {
		t.Fatalf("DeviceName = %q", found.DeviceName)
	}
	if len(found.PrivateAddrs) != 2 {
		t.Fatalf("len(PrivateAddrs) = %d, want 2", len(found.PrivateAddrs))
	}
}

func TestOnlineTableRemovesExpiredEntries(t *testing.T) {
	table := server.NewOnlineTable()

	table.Register(isig.OnlineEntry{DeviceID: "device-a", Fingerprint: "abc123"})

	if _, ok := table.Lookup("device-a"); !ok {
		t.Fatalf("device should be online")
	}

	table.PruneStale(-1 * time.Second)

	if _, ok := table.Lookup("device-a"); ok {
		t.Fatalf("device should have expired")
	}
}

func TestOnlineRegisterAndCandidateExchange(t *testing.T) {
	store := server.NewMemoryStore()
	table := server.NewOnlineTable()
	relayMgr := server.NewRelayManager(server.RelayConfig{Enabled: false})
	handler := server.NewSignalingServer(store, table, relayMgr)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	entry := isig.OnlineEntry{
		DeviceID:     "device-a",
		DeviceName:   "desktop-a",
		Fingerprint:  "abc123",
		ListenPort:   9001,
		PrivateAddrs: []string{"10.0.0.5:9001"},
	}
	body, _ := json.Marshal(entry)
	resp, err := http.Post(ts.URL+"/signaling/online", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("online register failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("online status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	getResp, err := http.Get(ts.URL + "/signaling/candidates?device_id=device-a")
	if err != nil {
		t.Fatalf("candidates fetch failed: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("candidates status = %d, want %d", getResp.StatusCode, http.StatusOK)
	}
}
