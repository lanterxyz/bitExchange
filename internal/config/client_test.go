package config_test

import (
    "path/filepath"
    "testing"

    "bitExchange/internal/config"
)

func TestSaveAndLoadClientConfigRoundTrips(t *testing.T) {
    path := filepath.Join(t.TempDir(), "config.json")
    cfg := config.ClientConfig{
        ServerURL:             "wss://example.com",
        RelayEnabled:          true,
        RelayMaxBytes:         67108864,
        ListenPort:            9001,
        LastPath:              "lan-direct",
        DeviceName:            "desktop-a",
        DefaultSaveRoot:       "/home/user/bitExchange",
        EncryptDefault:        false,
        TrustedDevicesVersion: 3,
    }

    if err := config.SaveClientConfig(path, cfg); err != nil {
        t.Fatalf("SaveClientConfig returned error: %v", err)
    }

    loaded, err := config.LoadClientConfig(path)
    if err != nil {
        t.Fatalf("LoadClientConfig returned error: %v", err)
    }

    if loaded.ServerURL != "wss://example.com" {
        t.Fatalf("ServerURL = %q", loaded.ServerURL)
    }
    if loaded.RelayMaxBytes != 67108864 {
        t.Fatalf("RelayMaxBytes = %d", loaded.RelayMaxBytes)
    }
    if loaded.DeviceName != "desktop-a" {
        t.Fatalf("DeviceName = %q", loaded.DeviceName)
    }
    if loaded.DefaultSaveRoot != "/home/user/bitExchange" {
        t.Fatalf("DefaultSaveRoot = %q", loaded.DefaultSaveRoot)
    }
    if loaded.TrustedDevicesVersion != 3 {
        t.Fatalf("TrustedDevicesVersion = %d", loaded.TrustedDevicesVersion)
    }
}
