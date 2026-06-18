package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitExchange/internal/coreapi"
)

func TestGetTasksListsCreatedTasks(t *testing.T) {
	srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: t.TempDir()})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"to_device_ids": []string{"device-a"},
		"body":          "x",
	})
	http.Post(ts.URL+"/api/send/text", "application/json", bytes.NewReader(body))

	resp, _ := http.Get(ts.URL + "/api/tasks")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var result struct {
		Tasks []coreapi.Task `json:"tasks"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result.Tasks))
	}
}
