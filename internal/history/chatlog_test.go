package history_test

import (
    "os"
    "path/filepath"
    "strings"
    "testing"

    "bitExchange/internal/history"
)

func TestAppendMessageWritesLineToChatFile(t *testing.T) {
    path := filepath.Join(t.TempDir(), "chat.txt")

    if err := history.AppendMessage(path, "[2026-06-09 10:00:00] desktop-a -> laptop-b: hello"); err != nil {
        t.Fatalf("AppendMessage returned error: %v", err)
    }

    data, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("ReadFile returned error: %v", err)
    }

    if !strings.Contains(string(data), "desktop-a -> laptop-b: hello") {
        t.Fatalf("chat.txt = %q", string(data))
    }
}
