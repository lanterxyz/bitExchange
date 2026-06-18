package coreapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitExchange/internal/coreapi"
)

func TestWriteErrorProducesStructuredJSON(t *testing.T) {
	w := httptest.NewRecorder()
	coreapi.WriteError(w, http.StatusServiceUnavailable, "path_unavailable", "no path to device-b")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Error.Code != "path_unavailable" {
		t.Fatalf("code = %q", body.Error.Code)
	}
	if body.Error.Message != "no path to device-b" {
		t.Fatalf("message = %q", body.Error.Message)
	}
}
