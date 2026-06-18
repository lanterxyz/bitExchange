package pairing

import (
    "encoding/json"
    "os"
)

type TrustedPeer struct {
    DeviceID    string `json:"device_id"`
    DeviceName  string `json:"device_name"`
    Fingerprint string `json:"fingerprint"`
}

type PeerStore struct {
    path string
}

func NewPeerStore(path string) *PeerStore {
    return &PeerStore{path: path}
}

func (s *PeerStore) LoadAll() ([]TrustedPeer, error) {
    data, err := os.ReadFile(s.path)
    if os.IsNotExist(err) {
        return []TrustedPeer{}, nil
    }
    if err != nil {
        return nil, err
    }

    var peers []TrustedPeer
    if err := json.Unmarshal(data, &peers); err != nil {
        return nil, err
    }

    return peers, nil
}

func (s *PeerStore) Save(peer TrustedPeer) error {
    peers, err := s.LoadAll()
    if err != nil {
        return err
    }

    peers = append(peers, peer)
    data, err := json.MarshalIndent(peers, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(s.path, data, 0o600)
}
