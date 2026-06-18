package coreapi

import (
	"net/http"
	"strings"
)

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET only")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]interface{}{"tasks": s.taskMgr.List()})
}

func (s *Server) handleTaskEvents(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/tasks/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		WriteError(w, http.StatusBadRequest, "missing_task_id", "task id required")
		return
	}
	id := parts[0]
	task, ok := s.taskMgr.Get(id)
	if !ok {
		WriteError(w, http.StatusNotFound, "task_not_found", "no such task")
		return
	}

	if len(parts) >= 2 && parts[1] == "cancel" {
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST only")
			return
		}
		s.taskMgr.Cancel(id)
		WriteJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
		return
	}

	if len(parts) >= 2 && parts[1] == "events" {
		broker := s.taskMgr.Broker(id)
		if broker == nil {
			WriteError(w, http.StatusNotFound, "no_broker", "task has no broker")
			return
		}
		writeSSE(w, r, broker)
		return
	}

	WriteJSON(w, http.StatusOK, task)
}

func writeSSE(w http.ResponseWriter, r *http.Request, broker *SSEBroker) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, http.StatusInternalServerError, "no_flusher", "streaming unsupported")
		return
	}
	sub, cancel := broker.Subscribe()
	defer cancel()
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev, open := <-sub:
			if !open {
				return
			}
			_, _ = w.Write([]byte("event: " + ev.Event + "\n"))
			_, _ = w.Write([]byte("data: " + ev.Data + "\n\n"))
			flusher.Flush()
		}
	}
}
