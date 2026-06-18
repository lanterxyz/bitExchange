package signaling_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"bitExchange/internal/signaling"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	aPub, aPriv, _ := ed25519.GenerateKey(rand.Reader)
	bPub, bPriv, _ := ed25519.GenerateKey(rand.Reader)

	codecA := signaling.NewCodec(aPub, aPriv)
	plain := []byte(`{"type":"p2p_offer","sdp":"test offer data"}`)

	envelope, err := codecA.Encrypt(plain, "device-a", "device-b", bPub)
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}

	codecB := signaling.NewCodec(bPub, bPriv)
	decrypted, err := codecB.Decrypt(envelope, aPub)
	if err != nil {
		t.Fatalf("Decrypt returned error: %v", err)
	}

	if string(decrypted) != string(plain) {
		t.Fatalf("decrypted = %q, want %q", decrypted, plain)
	}
}

func TestDecryptRejectsWrongSender(t *testing.T) {
	aPub, aPriv, _ := ed25519.GenerateKey(rand.Reader)
	bPub, bPriv, _ := ed25519.GenerateKey(rand.Reader)
	wrongPub, _, _ := ed25519.GenerateKey(rand.Reader)

	codecA := signaling.NewCodec(aPub, aPriv)
	envelope, _ := codecA.Encrypt([]byte("test"), "device-a", "device-b", bPub)

	// Use wrong key to try decrypting
	codecWrong := signaling.NewCodec(bPub, bPriv)
	_, err := codecWrong.Decrypt(envelope, wrongPub)
	if err == nil {
		t.Fatalf("Decrypt should have failed with wrong sender key")
	}
}

func TestEnvelopeContainsRequiredFields(t *testing.T) {
	aPub, aPriv, _ := ed25519.GenerateKey(rand.Reader)
	bPub, _, _ := ed25519.GenerateKey(rand.Reader)

	codec := signaling.NewCodec(aPub, aPriv)
	envelope, err := codec.Encrypt([]byte("test"), "device-a", "device-b", bPub)
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}

	if envelope.FromDeviceID != "device-a" {
		t.Fatalf("FromDeviceID = %q", envelope.FromDeviceID)
	}
	if envelope.ToDeviceID != "device-b" {
		t.Fatalf("ToDeviceID = %q", envelope.ToDeviceID)
	}
	if len(envelope.MessageID) == 0 {
		t.Fatalf("MessageID should not be empty")
	}
	if envelope.Timestamp == 0 {
		t.Fatalf("Timestamp should not be zero")
	}
	if len(envelope.Nonce) == 0 {
		t.Fatalf("Nonce should not be empty")
	}
	if len(envelope.Ciphertext) == 0 {
		t.Fatalf("Ciphertext should not be empty")
	}
	if len(envelope.Signature) == 0 {
		t.Fatalf("Signature should not be empty")
	}
}
