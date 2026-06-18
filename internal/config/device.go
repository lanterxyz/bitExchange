package config

import (
    "encoding/json"
    "os"

    icrypto "bitExchange/internal/crypto"
)

type DeviceConfig struct {
    Identity icrypto.Identity `json:"identity"`
}

func SaveDeviceConfig(path string, cfg DeviceConfig) error {
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0o600)
}

func LoadDeviceConfig(path string) (DeviceConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return DeviceConfig{}, err
    }

    var cfg DeviceConfig
    if err := json.Unmarshal(data, &cfg); err != nil {
        return DeviceConfig{}, err
    }

    return cfg, nil
}
