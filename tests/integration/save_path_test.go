package integration_test

import (
    "os"
    "path/filepath"
    "testing"

    "bitExchange/internal/config"
)

func TestConfiguredRootContainsChatFileAndReceivedFilesDirectory(t *testing.T) {
    root := filepath.Join(t.TempDir(), "my-bitexchange-root")

    paths, err := config.EnsureDefaultPaths(root)
    if err != nil {
        t.Fatalf("EnsureDefaultPaths returned error: %v", err)
    }

    if _, err := os.Stat(paths.ChatFile); err != nil {
        t.Fatalf("chat.txt missing: %v", err)
    }

    if stat, err := os.Stat(paths.ReceivedFilesDir); err != nil || !stat.IsDir() {
        t.Fatalf("receivedFiles directory invalid: %v", err)
    }
}
