package server_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "bitExchange/internal/server"
)

func TestRegisterAndFetchPairingCode(t *testing.T) {
    srv := server.NewHTTPServer(server.NewMemoryStore())
    ts := httptest.NewServer(srv)
    defer ts.Close()

    payload := map[string]string{
        "code": "PAIR-123456",
        "device_id": "device-a",
        "device_name": "desktop-a",
        "fingerprint": "abc123",
        "listen_addr": "127.0.0.1:9001",
    }

    body, _ := json.Marshal(payload)
    resp, err := http.Post(ts.URL+"/pairing/register", "application/json", bytes.NewReader(body))
    if err != nil {
        t.Fatalf("register request failed: %v", err)
    }
    if resp.StatusCode != http.StatusCreated {
        t.Fatalf("register status = %d, want %d", resp.StatusCode, http.StatusCreated)
    }

    getResp, err := http.Get(ts.URL + "/pairing/code/PAIR-123456")
    if err != nil {
        t.Fatalf("fetch request failed: %v", err)
    }
    if getResp.StatusCode != http.StatusOK {
        t.Fatalf("fetch status = %d, want %d", getResp.StatusCode, http.StatusOK)
    }
}
