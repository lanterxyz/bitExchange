package coreapi

import "net/http"

func (s *Server) handleInboxEvents(w http.ResponseWriter, r *http.Request) {
	writeSSE(w, r, s.inbox)
}
