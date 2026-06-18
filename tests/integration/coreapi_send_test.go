package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitExchange/internal/coreapi"
)

func TestSendTextCreatesPerTargetTasks(t *testing.T) {
	srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: t.TempDir()})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"to_device_ids": []string{"device-a", "device-b"},
		"body":          "hello",
		"encrypted":     false,
	})
	resp, err := http.Post(ts.URL+"/api/send/text", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var result struct {
		Tasks []struct {
			TaskID   string `json:"task_id"`
			ToDevice string `json:"to_device"`
		} `json:"tasks"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(result.Tasks))
	}
}
