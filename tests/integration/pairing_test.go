package integration_test

import (
    "testing"

    "bitExchange/internal/pairing"
)

func TestTrustedPeerRecordContainsFingerprintAndDeviceName(t *testing.T) {
    peer := pairing.TrustedPeer{
        DeviceID:    "device-b",
        DeviceName:  "laptop-b",
        Fingerprint: "fingerprint-b",
    }

    if peer.DeviceName != "laptop-b" {
        t.Fatalf("DeviceName = %q", peer.DeviceName)
    }

    if peer.Fingerprint != "fingerprint-b" {
        t.Fatalf("Fingerprint = %q", peer.Fingerprint)
    }
}
