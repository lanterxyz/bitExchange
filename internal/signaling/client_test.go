package signaling_test

import (
	"testing"
	"time"

	"bitExchange/internal/signaling"
)

func TestSignalingClientDialFailsOnInvalidURL(t *testing.T) {
	client := signaling.NewClient(signaling.ClientConfig{
		ServerURL:   "ws://127.0.0.1:19999",
		DeviceID:    "device-a",
		DeviceName:  "desktop-a",
		Fingerprint: "abc123",
	})

	err := client.Connect(1 * time.Second)
	if err == nil {
		t.Fatalf("Connect should have failed on invalid URL")
	}
}

func TestOnlineRegisterPayloadContainsPrivateAddrs(t *testing.T) {
	cfg := signaling.ClientConfig{
		ServerURL:     "ws://127.0.0.1:1",
		DeviceID:      "device-a",
		DeviceName:    "desktop-a",
		Fingerprint:   "abc123",
		ListenPort:    9001,
		PrivateAddrs:  []string{"10.0.0.5:9001", "192.168.1.10:9001"},
		RelayEnabled:  true,
		RelayMaxBytes: 67108864,
	}
	client := signaling.NewClient(cfg)

	got := client.Config()
	if got.PrivateAddrs[0] != "10.0.0.5:9001" {
		t.Fatalf("private addr[0] = %q, want %q", got.PrivateAddrs[0], "10.0.0.5:9001")
	}
	if got.RelayMaxBytes != 67108864 {
		t.Fatalf("RelayMaxBytes = %d", got.RelayMaxBytes)
	}
}

func TestClientConfigDefaults(t *testing.T) {
	client := signaling.NewClient(signaling.ClientConfig{
		DeviceID:    "device-x",
		Fingerprint: "fp-x",
	})

	cfg := client.Config()
	if cfg.DeviceID != "device-x" {
		t.Fatalf("DeviceID = %q", cfg.DeviceID)
	}
	if cfg.RelayMaxBytes != 0 {
		t.Fatalf("RelayMaxBytes default should be 0")
	}
}
