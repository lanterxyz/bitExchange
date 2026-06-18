package pairing_test

import (
    "path/filepath"
    "testing"

    "bitExchange/internal/pairing"
)

func TestPeerStoreSavesAndLoadsTrustedPeers(t *testing.T) {
    path := filepath.Join(t.TempDir(), "trusted-peers.json")
    store := pairing.NewPeerStore(path)

    peer := pairing.TrustedPeer{
        DeviceID:    "device-b",
        DeviceName:  "laptop-b",
        Fingerprint: "fingerprint-b",
    }

    if err := store.Save(peer); err != nil {
        t.Fatalf("Save returned error: %v", err)
    }

    peers, err := store.LoadAll()
    if err != nil {
        t.Fatalf("LoadAll returned error: %v", err)
    }

    if len(peers) != 1 {
        t.Fatalf("len(peers) = %d, want 1", len(peers))
    }

    if peers[0].DeviceID != peer.DeviceID {
        t.Fatalf("DeviceID = %q, want %q", peers[0].DeviceID, peer.DeviceID)
    }
}
