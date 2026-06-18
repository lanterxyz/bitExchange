package coreapi

import "net/http"

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, []interface{}{})
}
