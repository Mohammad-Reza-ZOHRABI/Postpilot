package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/config"
	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/mail"
)

type sendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
	HTML    string   `json:"html"`
	ReplyTo string   `json:"reply_to"`
}

type sendResponse struct {
	ID      int64  `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// APISend handles POST /api/v1/send.
func (h *Handler) APISend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keyID, err := h.ValidateAPIKey(r)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.From == "" || len(req.To) == 0 || req.Subject == "" {
		h.jsonError(w, "from, to, and subject are required", http.StatusBadRequest)
		return
	}

	if req.Text == "" && req.HTML == "" {
		h.jsonError(w, "text or html body is required", http.StatusBadRequest)
		return
	}

	// Log the email
	logID, _ := h.DB.LogEmail(&keyID, req.From, req.To[0], req.Subject, "queued")

	// Send via SMTP
	msg := &mail.Message{
		From:    req.From,
		To:      req.To,
		Subject: req.Subject,
		Text:    req.Text,
		HTML:    req.HTML,
		ReplyTo: req.ReplyTo,
	}

	if err := h.Mail.Send(msg); err != nil {
		_ = h.DB.UpdateEmailStatus(logID, "failed", "", err.Error())
		h.jsonError(w, "Failed to send: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = h.DB.UpdateEmailStatus(logID, "sent", "", "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(sendResponse{
		ID:     logID,
		Status: "sent",
	})
}

// APIHealth handles GET /api/v1/health.
func (h *Handler) APIHealth(w http.ResponseWriter, r *http.Request) {
	services := config.CheckServices()
	allUp := true
	svcMap := make(map[string]bool)
	for _, s := range services {
		svcMap[s.Name] = s.Running
		if !s.Running {
			allUp = false
		}
	}

	status := "ok"
	if !allUp {
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":   status,
		"services": svcMap,
	})
}

// APIStatus handles GET /api/v1/status/:id.
func (h *Handler) APIStatus(w http.ResponseWriter, r *http.Request) {
	_, err := h.ValidateAPIKey(r)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Path[len("/api/v1/status/"):]
	var id int64
	for _, c := range idStr {
		if c < '0' || c > '9' {
			h.jsonError(w, "Invalid ID", http.StatusBadRequest)
			return
		}
		id = id*10 + int64(c-'0')
	}

	log, err := h.DB.GetEmailLog(id)
	if err != nil {
		h.jsonError(w, "Email not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(log)
}
