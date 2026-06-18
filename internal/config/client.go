package config

import (
    "encoding/json"
    "os"
)

type ClientConfig struct {
    ServerURL             string `json:"server_url"`
    RelayEnabled          bool   `json:"relay_enabled"`
    RelayMaxBytes         int64  `json:"relay_max_bytes"`
    ListenPort            int    `json:"listen_port"`
    LastPath              string `json:"last_path,omitempty"`
    DeviceName            string `json:"device_name"`
    DefaultSaveRoot       string `json:"default_save_root"`
    EncryptDefault        bool   `json:"encrypt_default"`
    TrustedDevicesVersion int    `json:"trusted_devices_version"`
}

func SaveClientConfig(path string, cfg ClientConfig) error {
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0o600)
}

func LoadClientConfig(path string) (ClientConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return ClientConfig{}, nil
        }
        return ClientConfig{}, err
    }
    var cfg ClientConfig
    if err := json.Unmarshal(data, &cfg); err != nil {
        return ClientConfig{}, err
    }
    return cfg, nil
}
