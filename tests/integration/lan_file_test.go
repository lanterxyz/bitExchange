package integration_test

import (
    "bytes"
    "net"
    "os"
    "path/filepath"
    "testing"
    "time"

    "bitExchange/internal/transfer"
)

func TestLANFileSendSavesIntoReceivedFiles(t *testing.T) {
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatalf("Listen returned error: %v", err)
    }
    defer listener.Close()

    receivedDir := filepath.Join(t.TempDir(), "receivedFiles")
    done := make(chan error, 1)

    go func() {
        done <- transfer.AcceptOneFile(listener, receivedDir)
    }()

    time.Sleep(100 * time.Millisecond)

    payload := []byte("phase1 file payload")
    if err := transfer.SendFile(listener.Addr().String(), "desktop-a", "sample.txt", bytes.NewReader(payload), int64(len(payload))); err != nil {
        t.Fatalf("SendFile returned error: %v", err)
    }

    if err := <-done; err != nil {
        t.Fatalf("AcceptOneFile returned error: %v", err)
    }

    data, err := os.ReadFile(filepath.Join(receivedDir, "sample.txt"))
    if err != nil {
        t.Fatalf("ReadFile returned error: %v", err)
    }

    if string(data) != string(payload) {
        t.Fatalf("saved file = %q, want %q", data, payload)
    }
}
