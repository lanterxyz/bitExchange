package transfer_test

import (
    "bytes"
    "os"
    "path/filepath"
    "testing"

    "bitExchange/internal/transfer"
)

func TestSaveIncomingFileWritesIntoReceivedFilesDirectory(t *testing.T) {
    dir := filepath.Join(t.TempDir(), "receivedFiles")
    payload := []byte("hello file")

    savedPath, err := transfer.SaveIncomingFile(dir, "note.txt", bytes.NewReader(payload))
    if err != nil {
        t.Fatalf("SaveIncomingFile returned error: %v", err)
    }

    data, err := os.ReadFile(savedPath)
    if err != nil {
        t.Fatalf("ReadFile returned error: %v", err)
    }

    if string(data) != "hello file" {
        t.Fatalf("file contents = %q", string(data))
    }
}
