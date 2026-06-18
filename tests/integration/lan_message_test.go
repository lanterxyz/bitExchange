package integration_test

import (
    "net"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "bitExchange/internal/history"
    "bitExchange/internal/transfer"
)

func TestLANTextSendAppendsToReceiverChatFile(t *testing.T) {
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatalf("Listen returned error: %v", err)
    }
    defer listener.Close()

    chatFile := filepath.Join(t.TempDir(), "chat.txt")
    done := make(chan error, 1)

    go func() {
        done <- transfer.AcceptOneTextMessage(listener, chatFile)
    }()

    time.Sleep(100 * time.Millisecond)

    if err := transfer.SendText(listener.Addr().String(), "desktop-a", "hello over lan"); err != nil {
        t.Fatalf("SendText returned error: %v", err)
    }

    if err := <-done; err != nil {
        t.Fatalf("AcceptOneTextMessage returned error: %v", err)
    }

    data, err := os.ReadFile(chatFile)
    if err != nil {
        t.Fatalf("ReadFile returned error: %v", err)
    }

    if !strings.Contains(string(data), "hello over lan") {
        t.Fatalf("chat.txt = %q", string(data))
    }

    _ = history.AppendMessage
}
