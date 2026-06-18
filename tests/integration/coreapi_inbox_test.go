package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bitExchange/internal/coreapi"
)

func TestInboxSSEEndpointConnects(t *testing.T) {
	srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: t.TempDir()})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/inbox/events")
	if err != nil {
		t.Fatalf("inbox events: %v", err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("content-type = %q", resp.Header.Get("Content-Type"))
	}
	time.Sleep(50 * time.Millisecond)
}
