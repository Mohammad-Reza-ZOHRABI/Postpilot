package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/database"
)

// APIGetKeys handles GET /api/v1/keys.
func (h *Handler) APIGetKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keys, err := h.DB.ListAPIKeys()
	if err != nil {
		h.jsonError(w, "Failed to list keys", http.StatusInternalServerError)
		return
	}
	if keys == nil {
		keys = []database.APIKey{}
	}

	h.jsonOK(w, map[string]any{"keys": keys})
}

// APICreateKey handles POST /api/v1/keys.
func (h *Handler) APICreateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Permissions string `json:"permissions"`
		RateLimit   int    `json:"rate_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		req.Name = "Unnamed key"
	}
	if req.Permissions == "" {
		req.Permissions = "send"
	}
	if req.RateLimit <= 0 {
		req.RateLimit = 100
	}

	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		h.jsonError(w, "Failed to generate key", http.StatusInternalServerError)
		return
	}

	keyStr := "pp_live_" + hex.EncodeToString(rawKey)
	keyPrefix := keyStr[:16] + "..."

	hash := sha256.Sum256([]byte(keyStr))
	keyHash := hex.EncodeToString(hash[:])

	if err := h.DB.CreateAPIKey(req.Name, keyHash, keyPrefix, req.Permissions, req.RateLimit); err != nil {
		h.jsonError(w, "Failed to save key: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.jsonOK(w, map[string]string{"key": keyStr})
}

// APIRevokeKey handles POST /api/v1/keys/{id}/revoke.
func (h *Handler) APIRevokeKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse ID from URL: /api/v1/keys/{id}/revoke
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/keys/")
	path = strings.TrimSuffix(path, "/revoke")
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		h.jsonError(w, "Invalid key ID", http.StatusBadRequest)
		return
	}

	if err := h.DB.RevokeAPIKey(id); err != nil {
		h.jsonError(w, "Failed to revoke key", http.StatusInternalServerError)
		return
	}

	h.jsonOK(w, map[string]bool{"success": true})
}

// ValidateAPIKey checks an API key from request headers and returns the key ID.
func (h *Handler) ValidateAPIKey(r *http.Request) (int64, error) {
	key := r.Header.Get("X-API-Key")
	if key == "" {
		bearer := r.Header.Get("Authorization")
		if len(bearer) > 7 && bearer[:7] == "Bearer " {
			key = bearer[7:]
		}
	}
	if key == "" {
		return 0, fmt.Errorf("missing API key")
	}

	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])

	apiKey, err := h.DB.GetAPIKeyByHash(keyHash)
	if err != nil {
		return 0, fmt.Errorf("invalid API key")
	}
	if apiKey == nil {
		return 0, fmt.Errorf("invalid API key")
	}
	if apiKey.RevokedAt != nil {
		return 0, fmt.Errorf("revoked API key")
	}

	_ = h.DB.IncrementAPIKeyUsage(apiKey.ID)
	return apiKey.ID, nil
}
