package config_test

import (
    "path/filepath"
    "testing"

    "bitExchange/internal/config"
)

func TestEnsureDefaultPathsCreatesChatLogAndReceivedFiles(t *testing.T) {
    root := filepath.Join(t.TempDir(), "bitExchange")

    paths, err := config.EnsureDefaultPaths(root)
    if err != nil {
        t.Fatalf("EnsureDefaultPaths returned error: %v", err)
    }

    if paths.RootDir != root {
        t.Fatalf("RootDir = %q, want %q", paths.RootDir, root)
    }

    if paths.ChatFile != filepath.Join(root, "chat.txt") {
        t.Fatalf("ChatFile = %q", paths.ChatFile)
    }

    if paths.ReceivedFilesDir != filepath.Join(root, "receivedFiles") {
        t.Fatalf("ReceivedFilesDir = %q", paths.ReceivedFilesDir)
    }
}
