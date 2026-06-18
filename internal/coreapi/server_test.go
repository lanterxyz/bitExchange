package coreapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bitExchange/internal/coreapi"
)

func TestGetConfigReturnsPersistedConfig(t *testing.T) {
	root := t.TempDir()
	srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: root})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/config")
	if err != nil {
		t.Fatalf("GET config: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var cfg map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&cfg)
	if cfg["device_name"] == nil {
		t.Fatalf("device_name missing")
	}
}

func TestPutConfigPersists(t *testing.T) {
	root := t.TempDir()
	srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: root})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := []byte(`{"device_name":"laptop-x","default_save_root":"/tmp/x","server_url":"wss://s","relay_enabled":true,"relay_max_bytes":1024,"listen_port":9001,"encrypt_default":false,"trusted_devices_version":0}`)
	resp, err := http.Post(ts.URL+"/api/config", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("PUT config: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	data, _ := os.ReadFile(filepath.Join(root, "config.json"))
	if !strings.Contains(string(data), "laptop-x") {
		t.Fatalf("config not persisted: %s", data)
	}
}
