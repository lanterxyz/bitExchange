package server

import (
    "errors"
    "sync"

    "bitExchange/internal/pairing"
)

var ErrCodeNotFound = errors.New("pairing code not found")

type MemoryStore struct {
    mu    sync.RWMutex
    codes map[string]pairing.CodeRecord
}

func NewMemoryStore() *MemoryStore {
    return &MemoryStore{codes: map[string]pairing.CodeRecord{}}
}

func (s *MemoryStore) Put(record pairing.CodeRecord) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.codes[record.Code] = record
}

func (s *MemoryStore) Get(code string) (pairing.CodeRecord, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    record, ok := s.codes[code]
    if !ok {
        return pairing.CodeRecord{}, ErrCodeNotFound
    }

    return record, nil
}
