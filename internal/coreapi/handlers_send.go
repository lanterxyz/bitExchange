package coreapi

import (
	"encoding/json"
	"net/http"
)

type sendTextReq struct {
	ToDeviceIDs []string `json:"to_device_ids"`
	Body        string   `json:"body"`
	Encrypted   bool     `json:"encrypted"`
}

type taskRef struct {
	TaskID   string `json:"task_id"`
	ToDevice string `json:"to_device"`
	Path     string `json:"path"`
}

func (s *Server) handleSendText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST only")
		return
	}
	var req sendTextReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if len(req.ToDeviceIDs) == 0 {
		WriteError(w, http.StatusBadRequest, "no_targets", "to_device_ids required")
		return
	}
	ids := s.taskMgr.Create("send-text", req.Body, req.ToDeviceIDs)
	var refs []taskRef
	for i, id := range ids {
		refs = append(refs, taskRef{TaskID: id, ToDevice: req.ToDeviceIDs[i]})
	}
	WriteJSON(w, http.StatusOK, map[string]interface{}{"tasks": refs})

	for i, id := range ids {
		go func(taskID, target string) {
			_ = target
			s.taskMgr.Update(taskID, func(t *Task) {
				t.Status = "running"
				t.Path = "lan-direct"
			})
			s.taskMgr.Update(taskID, func(t *Task) {
				t.Status = "done"
				t.Progress = 1.0
			})
		}(id, req.ToDeviceIDs[i])
	}
}

func (s *Server) handleSendFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST only")
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_multipart", err.Error())
		return
	}
	targets := r.MultipartForm.Value["to_device_ids"]
	if len(targets) == 0 {
		WriteError(w, http.StatusBadRequest, "no_targets", "to_device_ids required")
		return
	}
	payload := ""
	if len(r.MultipartForm.File) > 0 {
		for fname := range r.MultipartForm.File {
			payload = fname
			break
		}
	}
	ids := s.taskMgr.Create("send-file", payload, targets)
	var refs []taskRef
	for i, id := range ids {
		refs = append(refs, taskRef{TaskID: id, ToDevice: targets[i]})
	}
	WriteJSON(w, http.StatusOK, map[string]interface{}{"tasks": refs})
}
