package signaling

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"
)

// ed25519P is the prime for the edwards25519 / curve25519 field: 2^255 - 19.
var ed25519P = new(big.Int).SetBytes([]byte{
	0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xed,
})

// Codec encrypts and decrypts signaling messages between devices using
// X25519 ECDH + AES-256-GCM and ed25519 signatures.
type Codec struct {
	pubKey  ed25519.PublicKey
	privKey ed25519.PrivateKey
}

// NewCodec returns a Codec initialized with the device's ed25519 key pair.
func NewCodec(pubKey ed25519.PublicKey, privKey ed25519.PrivateKey) *Codec {
	return &Codec{pubKey: pubKey, privKey: privKey}
}

func generateMessageID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ed25519PubToCurve25519 converts an ed25519 public key to a curve25519
// (X25519) public key using the birational equivalence between Ed25519
// and Curve25519: u = (1 + y) / (1 - y) mod p.
//
// Both ed25519 and X25519 keys are stored in little-endian per their
// respective RFCs (8032, 7748). math/big uses big-endian, so we reverse.
func ed25519PubToCurve25519(edPub ed25519.PublicKey) (*[32]byte, error) {
	yBytes := make([]byte, 32)
	copy(yBytes, edPub)
	yBytes[31] &= 0x7F // clear sign bit

	// Reverse little-endian to big-endian for math/big.
	reverseBytes(yBytes)

	y := new(big.Int).SetBytes(yBytes)
	one := big.NewInt(1)
	p := ed25519P

	// u = (1 + y) * (1 - y)^{-1} mod p
	num := new(big.Int).Add(one, y)
	num.Mod(num, p)
	denom := new(big.Int).Sub(one, y)
	denom.Mod(denom, p)
	if denom.Sign() == 0 {
		return nil, errors.New("invalid ed25519 public key (y = 1)")
	}
	denom.ModInverse(denom, p)
	u := new(big.Int).Mul(num, denom)
	u.Mod(u, p)

	uB := u.Bytes()
	// uB is big-endian; pad then reverse to little-endian for X25519.
	padded := make([]byte, 32)
	copy(padded[32-len(uB):], uB)
	reverseBytes(padded)

	var out [32]byte
	copy(out[:], padded)
	return &out, nil
}

func reverseBytes(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

// ed25519PrivToX25519 derives an X25519 private key from an ed25519
// private key. ed25519 uses SHA-512(seed)[:32] with clamping as the
// scalar; we must compute the same scalar for X25519 so that the
// curve25519 public key matches the birational-map-converted ed25519 key.
func ed25519PrivToX25519(edPriv ed25519.PrivateKey) (*ecdh.PrivateKey, error) {
	seed := edPriv.Seed()
	h := sha512.Sum512(seed)
	// First 32 bytes of SHA-512(seed) is the ed25519 scalar seed.
	scalar := h[:32]
	curve := ecdh.X25519()
	return curve.NewPrivateKey(scalar)
}

func deriveAESKey(shared []byte) []byte {
	h := sha256.Sum256(shared)
	return h[:]
}

// Encrypt encrypts plaintext for the given recipient and returns a signed envelope.
// It uses X25519 ECDH + AES-256-GCM for encryption and ed25519 for signing.
func (c *Codec) Encrypt(plain []byte, fromID, toID string, recipientPubKey ed25519.PublicKey) (SignedEnvelope, error) {
	// Convert sender's ed25519 private key to X25519 private key.
	senderXPriv, err := ed25519PrivToX25519(c.privKey)
	if err != nil {
		return SignedEnvelope{}, fmt.Errorf("convert sender key: %w", err)
	}

	// Convert recipient's ed25519 public key to X25519 public key.
	recipXPubRaw, err := ed25519PubToCurve25519(recipientPubKey)
	if err != nil {
		return SignedEnvelope{}, fmt.Errorf("convert recipient key: %w", err)
	}
	recipXPub, err := ecdh.X25519().NewPublicKey(recipXPubRaw[:])
	if err != nil {
		return SignedEnvelope{}, fmt.Errorf("parse recipient curve key: %w", err)
	}

	// ECDH shared secret.
	shared, err := senderXPriv.ECDH(recipXPub)
	if err != nil {
		return SignedEnvelope{}, fmt.Errorf("ecdh: %w", err)
	}

	// Derive AES key from shared secret.
	aesKey := deriveAESKey(shared)

	// AES-256-GCM encrypt.
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return SignedEnvelope{}, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return SignedEnvelope{}, fmt.Errorf("gcm: %w", err)
	}

	// 12-byte nonce for AES-GCM.
	nonceSize := gcm.NonceSize()
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return SignedEnvelope{}, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plain, nil)

	envelope := SignedEnvelope{
		FromDeviceID: fromID,
		ToDeviceID:   toID,
		MessageID:    generateMessageID(),
		Timestamp:    time.Now().Unix(),
		Nonce:        nonce,
		Ciphertext:   ciphertext,
	}

	// Sign the envelope metadata + nonce + ciphertext.
	sigPayload := append([]byte(fromID+toID+envelope.MessageID), nonce...)
	sigPayload = append(sigPayload, ciphertext...)
	envelope.Signature = ed25519.Sign(c.privKey, sigPayload)

	return envelope, nil
}

// Decrypt verifies the envelope signature and decrypts the ciphertext.
func (c *Codec) Decrypt(env SignedEnvelope, senderPubKey ed25519.PublicKey) ([]byte, error) {
	// Verify signature.
	sigPayload := append([]byte(env.FromDeviceID+env.ToDeviceID+env.MessageID), env.Nonce...)
	sigPayload = append(sigPayload, env.Ciphertext...)
	if !ed25519.Verify(senderPubKey, sigPayload, env.Signature) {
		return nil, errors.New("signature verification failed")
	}

	// Convert recipient's ed25519 private key to X25519 private key.
	recipXPriv, err := ed25519PrivToX25519(c.privKey)
	if err != nil {
		return nil, fmt.Errorf("convert recipient key: %w", err)
	}

	// Convert sender's ed25519 public key to X25519 public key.
	senderXPubRaw, err := ed25519PubToCurve25519(senderPubKey)
	if err != nil {
		return nil, fmt.Errorf("convert sender key: %w", err)
	}
	senderXPub, err := ecdh.X25519().NewPublicKey(senderXPubRaw[:])
	if err != nil {
		return nil, fmt.Errorf("parse sender curve key: %w", err)
	}

	// ECDH shared secret.
	shared, err := recipXPriv.ECDH(senderXPub)
	if err != nil {
		return nil, fmt.Errorf("ecdh: %w", err)
	}

	// Derive AES key.
	aesKey := deriveAESKey(shared)

	// AES-256-GCM decrypt.
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	plain, err := gcm.Open(nil, env.Nonce, env.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plain, nil
}
