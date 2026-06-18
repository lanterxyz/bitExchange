package config_test

import (
    "path/filepath"
    "testing"

    "bitExchange/internal/config"
    icrypto "bitExchange/internal/crypto"
)

func TestSaveAndLoadDeviceConfigRoundTripsIdentity(t *testing.T) {
    root := t.TempDir()
    identity, err := icrypto.GenerateIdentity("desktop-a")
    if err != nil {
        t.Fatalf("GenerateIdentity returned error: %v", err)
    }

    path := filepath.Join(root, "device.json")
    if err := config.SaveDeviceConfig(path, config.DeviceConfig{Identity: identity}); err != nil {
        t.Fatalf("SaveDeviceConfig returned error: %v", err)
    }

    loaded, err := config.LoadDeviceConfig(path)
    if err != nil {
        t.Fatalf("LoadDeviceConfig returned error: %v", err)
    }

    if loaded.Identity.DeviceID != identity.DeviceID {
        t.Fatalf("DeviceID = %q, want %q", loaded.Identity.DeviceID, identity.DeviceID)
    }
}
