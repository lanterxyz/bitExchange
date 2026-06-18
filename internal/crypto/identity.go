package crypto

import (
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
)

type Identity struct {
    DeviceName  string `json:"device_name"`
    DeviceID    string `json:"device_id"`
    Fingerprint string `json:"fingerprint"`
    PublicKey   []byte `json:"public_key"`
    PrivateKey  []byte `json:"private_key"`
}

func GenerateIdentity(deviceName string) (Identity, error) {
    publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return Identity{}, err
    }

    sum := sha256.Sum256(publicKey)

    return Identity{
        DeviceName:  deviceName,
        DeviceID:    hex.EncodeToString(sum[:16]),
        Fingerprint: hex.EncodeToString(sum[:]),
        PublicKey:   publicKey,
        PrivateKey:  privateKey,
    }, nil
}
