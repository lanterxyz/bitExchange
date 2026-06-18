package crypto_test

import (
    "testing"

    "bitExchange/internal/crypto"
)

func TestGenerateIdentityProducesStableFingerprintFormat(t *testing.T) {
    identity, err := crypto.GenerateIdentity("work-laptop")
    if err != nil {
        t.Fatalf("GenerateIdentity returned error: %v", err)
    }

    if identity.DeviceName != "work-laptop" {
        t.Fatalf("DeviceName = %q", identity.DeviceName)
    }

    if len(identity.DeviceID) == 0 {
        t.Fatalf("DeviceID should not be empty")
    }

    if len(identity.Fingerprint) < 16 {
        t.Fatalf("Fingerprint too short: %q", identity.Fingerprint)
    }
}
