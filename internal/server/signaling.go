package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	isig "bitExchange/internal/signaling"
)

type OnlineTable struct {
	mu      sync.RWMutex
	entries map[string]isig.OnlineEntry
}

func NewOnlineTable() *OnlineTable {
	return &OnlineTable{entries: map[string]isig.OnlineEntry{}}
}

func (t *OnlineTable) Register(entry isig.OnlineEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry.RegisteredAt = time.Now()
	t.entries[entry.DeviceID] = entry
}

func (t *OnlineTable) Lookup(deviceID string) (isig.OnlineEntry, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	entry, ok := t.entries[deviceID]
	return entry, ok
}

func (t *OnlineTable) PruneStale(maxAge time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for id, entry := range t.entries {
		if entry.RegisteredAt.Before(cutoff) {
			delete(t.entries, id)
		}
	}
}

// SignalingServer combines pairing, signaling, and relay routes.
type SignalingServer struct {
	store    *MemoryStore
	table    *OnlineTable
	relayMgr *RelayManager
	mux      *http.ServeMux
}

func NewSignalingServer(store *MemoryStore, table *OnlineTable, relayMgr *RelayManager) http.Handler {
	srv := &SignalingServer{
		store:    store,
		table:    table,
		relayMgr: relayMgr,
		mux:      http.NewServeMux(),
	}
	srv.routes()
	return srv.mux
}

func (s *SignalingServer) routes() {
	s.mux.HandleFunc("/pairing/register", HandlePairingRegister(s.store))
	s.mux.HandleFunc("/pairing/code/", HandlePairingFetch(s.store))
	s.mux.HandleFunc("/signaling/online", s.handleOnline)
	s.mux.HandleFunc("/signaling/candidates", s.handleCandidates)
	s.mux.HandleFunc("/relay/create", s.handleRelayCreate)
	s.mux.HandleFunc("/relay/send", s.handleRelaySend)
}

func (s *SignalingServer) handleOnline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var entry isig.OnlineEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.table.Register(entry)
	w.WriteHeader(http.StatusOK)
}

func (s *SignalingServer) handleCandidates(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	entry, ok := s.table.Lookup(deviceID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func (s *SignalingServer) handleRelayCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		FromDevice string `json:"from_device"`
		ToDevice   string `json:"to_device"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionID, err := s.relayMgr.CreateSession(req.FromDevice, req.ToDevice)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"session_id": sessionID})
}

func (s *SignalingServer) handleRelaySend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Kind      string `json:"kind"`
		SessionID string `json:"session_id"`
		Body      string `json:"body"`
		FileSize  int64  `json:"file_size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	session, ok := s.relayMgr.GetSession(req.SessionID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "session not found"})
		return
	}

	relayCfg := s.relayMgr.Config()
	dataLen := int64(len(req.Body))
	if req.Kind == "relay_file" {
		dataLen = req.FileSize
	}
	if dataLen > relayCfg.MaxBytes {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("payload exceeds relay limit of %d bytes", relayCfg.MaxBytes),
		})
		return
	}

	// For now, relay is a pass-through: data is forwarded in the response
	// Full duplex relay will be implemented with WebSocket later
	_ = session
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "delivered"})
}
