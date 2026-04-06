package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/auth"
)

// APIListUsers handles GET /api/v1/users (admin only).
func (h *Handler) APIListUsers(w http.ResponseWriter, r *http.Request) {
	claims := h.authenticateFromCookie(r)
	if claims == nil || claims.Role != "admin" {
		h.jsonError(w, "Admin access required", http.StatusForbidden)
		return
	}

	users, err := h.DB.ListUsers()
	if err != nil {
		h.jsonError(w, "Failed to list users", http.StatusInternalServerError)
		return
	}

	type userResp struct {
		ID        int64  `json:"id"`
		Email     string `json:"email"`
		Role      string `json:"role"`
		CreatedAt string `json:"created_at"`
	}

	var list []userResp
	for _, u := range users {
		list = append(list, userResp{
			ID:        u.ID,
			Email:     u.Email,
			Role:      u.Role,
			CreatedAt: u.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	if list == nil {
		list = []userResp{}
	}

	h.jsonOK(w, map[string]any{"users": list})
}

// APICreateUser handles POST /api/v1/users (admin only).
func (h *Handler) APICreateUser(w http.ResponseWriter, r *http.Request) {
	claims := h.authenticateFromCookie(r)
	if claims == nil || claims.Role != "admin" {
		h.jsonError(w, "Admin access required", http.StatusForbidden)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(req.Email)
	if email == "" || req.Password == "" {
		h.jsonError(w, "Email and password are required", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 12 {
		h.jsonError(w, "Password must be at least 12 characters", http.StatusBadRequest)
		return
	}
	role := req.Role
	if role != "admin" && role != "member" {
		role = "member"
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.jsonError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	secret, url, err := auth.GenerateTOTP(email)
	if err != nil {
		h.jsonError(w, "Failed to generate 2FA", http.StatusInternalServerError)
		return
	}

	if err := h.DB.CreateUser(email, hash, secret, role); err != nil {
		h.jsonError(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	qrDataURL := generateQRDataURL(url)

	h.jsonOK(w, map[string]any{
		"totp_secret":  secret,
		"qr_data_url":  qrDataURL,
	})
}

// APIDeleteUser handles POST /api/v1/users/{id}/delete (admin only).
func (h *Handler) APIDeleteUser(w http.ResponseWriter, r *http.Request) {
	claims := h.authenticateFromCookie(r)
	if claims == nil || claims.Role != "admin" {
		h.jsonError(w, "Admin access required", http.StatusForbidden)
		return
	}

	// Extract ID from path: /api/v1/users/123/delete
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		h.jsonError(w, "Invalid path", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		h.jsonError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if id == claims.UserID {
		h.jsonError(w, "Cannot delete yourself", http.StatusBadRequest)
		return
	}

	if err := h.DB.DeleteUser(id); err != nil {
		h.jsonError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	h.jsonOK(w, map[string]bool{"success": true})
}

// APIUpdateUserRole handles POST /api/v1/users/{id}/role (admin only).
func (h *Handler) APIUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	claims := h.authenticateFromCookie(r)
	if claims == nil || claims.Role != "admin" {
		h.jsonError(w, "Admin access required", http.StatusForbidden)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		h.jsonError(w, "Invalid path", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		h.jsonError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Role != "admin" && req.Role != "member" {
		h.jsonError(w, "Role must be 'admin' or 'member'", http.StatusBadRequest)
		return
	}

	if err := h.DB.UpdateUserRole(id, req.Role); err != nil {
		h.jsonError(w, "Failed to update role", http.StatusInternalServerError)
		return
	}

	h.jsonOK(w, map[string]bool{"success": true})
}
