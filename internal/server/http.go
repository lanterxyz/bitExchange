package server

import (
    "encoding/json"
    "net/http"

    "bitExchange/internal/pairing"
)

type HTTPServer struct {
    store *MemoryStore
    mux   *http.ServeMux
}

func NewHTTPServer(store *MemoryStore) http.Handler {
    srv := &HTTPServer{store: store, mux: http.NewServeMux()}
    srv.routes()
    return srv.mux
}

func (s *HTTPServer) routes() {
    s.mux.HandleFunc("/pairing/register", s.handleRegister)
    s.mux.HandleFunc("/pairing/code/", s.handleFetch)
}

func (s *HTTPServer) handleRegister(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    var record pairing.CodeRecord
    if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    s.store.Put(record)
    w.WriteHeader(http.StatusCreated)
}

func (s *HTTPServer) handleFetch(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Path[len("/pairing/code/"):]
    record, err := s.store.Get(code)
    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(record)
}

// Exported standalone handler functions for reuse by SignalingServer.

func HandlePairingRegister(store *MemoryStore) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }
        var record pairing.CodeRecord
        if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
            w.WriteHeader(http.StatusBadRequest)
            return
        }
        store.Put(record)
        w.WriteHeader(http.StatusCreated)
    }
}

func HandlePairingFetch(store *MemoryStore) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        code := r.URL.Path[len("/pairing/code/"):]
        record, err := store.Get(code)
        if err != nil {
            w.WriteHeader(http.StatusNotFound)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(record)
    }
}
