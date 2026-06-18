package coreapi

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"

	"bitExchange/internal/config"
)

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := s.loadConfig()
		WriteJSON(w, http.StatusOK, cfg)
	case http.MethodPost, http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
		var cfg config.ClientConfig
		if err := json.Unmarshal(body, &cfg); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if err := AtomicWrite(filepath.Join(s.cfg.RootDir, "config.json"), body); err != nil {
			WriteError(w, http.StatusInternalServerError, "write_failed", err.Error())
			return
		}
		WriteJSON(w, http.StatusOK, cfg)
	default:
		WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "only GET/PUT allowed")
	}
}

func (s *Server) handleDevice(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{
		"device_id":   "device-local",
		"device_name": s.loadConfig().DeviceName,
		"fingerprint": "fp-placeholder",
	})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	limit := 100
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"lines": []string{},
		"limit": limit,
	})
}
