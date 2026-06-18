package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitExchange/internal/coreapi"
)

func TestPeersReturnsEmptyListByDefault(t *testing.T) {
	srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: t.TempDir()})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, _ := http.Get(ts.URL + "/api/peers")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var peers []interface{}
	json.NewDecoder(resp.Body).Decode(&peers)
	if len(peers) != 0 {
		t.Fatalf("expected empty peers list, got %d", len(peers))
	}
}
