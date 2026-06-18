package integration_test

import (
    "crypto/ed25519"
    "crypto/rand"
    "testing"

    "bitExchange/internal/signaling"
)

func TestEncryptedSignalRoundTripEndToEnd(t *testing.T) {
    aPub, aPriv, _ := ed25519.GenerateKey(rand.Reader)
    bPub, bPriv, _ := ed25519.GenerateKey(rand.Reader)

    codecA := signaling.NewCodec(aPub, aPriv)
    codecB := signaling.NewCodec(bPub, bPriv)

    plain := []byte(`{"type":"p2p_offer","sdp":"v=0..."}`)
    envelope, err := codecA.Encrypt(plain, "device-a", "device-b", bPub)
    if err != nil {
        t.Fatalf("Encrypt error: %v", err)
    }

    decrypted, err := codecB.Decrypt(envelope, aPub)
    if err != nil {
        t.Fatalf("Decrypt error: %v", err)
    }

    if string(decrypted) != string(plain) {
        t.Fatalf("decrypted = %q, want %q", decrypted, plain)
    }

    if envelope.FromDeviceID != "device-a" {
        t.Fatalf("envelope FromDeviceID = %q", envelope.FromDeviceID)
    }
    if envelope.ToDeviceID != "device-b" {
        t.Fatalf("envelope ToDeviceID = %q", envelope.ToDeviceID)
    }
    if len(envelope.MessageID) == 0 {
        t.Fatalf("envelope MessageID should not be empty")
    }
}

func TestEncryptedSignalRejectsTamperedCiphertext(t *testing.T) {
    aPub, aPriv, _ := ed25519.GenerateKey(rand.Reader)
    bPub, bPriv, _ := ed25519.GenerateKey(rand.Reader)

    codecA := signaling.NewCodec(aPub, aPriv)
    codecB := signaling.NewCodec(bPub, bPriv)

    envelope, _ := codecA.Encrypt([]byte("secret"), "device-a", "device-b", bPub)

    // Tamper with ciphertext
    if len(envelope.Ciphertext) > 0 {
        envelope.Ciphertext[0] ^= 0xFF
    }

    _, err := codecB.Decrypt(envelope, aPub)
    if err == nil {
        t.Fatalf("Decrypt should have failed with tampered ciphertext")
    }
}
