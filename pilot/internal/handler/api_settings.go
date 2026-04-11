package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/config"
)

var settingsKeys = []string{
	"mail_mode", "postfix_hostname", "postfix_myorigin",
	"postfix_message_size", "postfix_mynetworks",
	"postfix_relay_host", "postfix_relay_user", "postfix_relay_pass",
	"dkim_enabled", "dkim_domain", "dkim_selector", "dkim_key_size",
	"mp_max_messages",
}

// APIGetSettings handles GET /api/v1/settings.
func (h *Handler) APIGetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	settings, _ := h.DB.GetAllSettings()
	if settings == nil {
		settings = make(map[string]string)
	}

	mode := settings["mail_mode"]
	if mode == "" {
		mode = "catch"
	}

	h.jsonOK(w, map[string]any{
		"settings": settings,
		"mode":     mode,
	})
}

// APISaveSettings handles POST /api/v1/settings.
func (h *Handler) APISaveSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.jsonError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	for _, key := range settingsKeys {
		if val, ok := body[key]; ok && val != "" {
			_ = h.DB.SetSetting(key, val)
		}
	}

	// Handle unchecked DKIM: if dkim_enabled is not in the body, set to "false"
	if _, ok := body["dkim_enabled"]; !ok {
		_ = h.DB.SetSetting("dkim_enabled", "false")
	}

	// Apply settings to Postfix
	settings, _ := h.DB.GetAllSettings()
	if err := config.ApplySettings(settings); err != nil {
		log.Printf("ApplySettings failed: %v", err)
		h.jsonError(w, "Settings saved but failed to apply", http.StatusInternalServerError)
		return
	}

	h.jsonOK(w, map[string]bool{"success": true})
}
