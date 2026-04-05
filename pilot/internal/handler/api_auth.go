package handler

import (
	"encoding/base64"
	"encoding/json"
	"image/png"
	"net/http"
	"strings"
	"time"

	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/auth"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// APIAuthCheck handles GET /api/v1/auth/check.
// Returns whether setup is needed and whether the user is logged in.
func (h *Handler) APIAuthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hasUsers, _ := h.DB.HasUsers()
	loggedIn := h.authenticateFromCookie(r) != nil

	h.jsonOK(w, map[string]bool{
		"setup_needed": !hasUsers,
		"logged_in":    loggedIn,
	})
}

// setupRequest represents the JSON body for the setup endpoint.
type setupRequest struct {
	Step       int    `json:"step"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	TOTPSecret string `json:"totp_secret"`
	TOTPCode   string `json:"totp_code"`
}

// APISetup handles POST /api/v1/auth/setup.
// Step 1: validate email+password, generate TOTP, return secret + QR.
// Step 2: validate TOTP code, create user, return success.
func (h *Handler) APISetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Block setup if users already exist
	hasUsers, _ := h.DB.HasUsers()
	if hasUsers {
		h.jsonError(w, "Setup already completed", http.StatusForbidden)
		return
	}

	var req setupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	step := req.Step
	if step == 0 {
		step = 1
	}

	email := strings.TrimSpace(req.Email)
	password := req.Password

	if step == 1 {
		if email == "" || password == "" {
			h.jsonError(w, "Email and password are required", http.StatusBadRequest)
			return
		}
		if len(password) < 12 {
			h.jsonError(w, "Password must be at least 12 characters", http.StatusBadRequest)
			return
		}

		secret, url, err := auth.GenerateTOTP(email)
		if err != nil {
			h.jsonError(w, "Failed to generate 2FA secret", http.StatusInternalServerError)
			return
		}

		qrDataURL := generateQRDataURL(url)

		h.jsonOK(w, map[string]string{
			"totp_secret":  secret,
			"qr_data_url":  qrDataURL,
		})
		return
	}

	if step == 2 {
		totpSecret := req.TOTPSecret
		totpCode := req.TOTPCode

		if email == "" || password == "" || totpSecret == "" || totpCode == "" {
			h.jsonError(w, "All fields are required for step 2", http.StatusBadRequest)
			return
		}

		if len(password) < 12 {
			h.jsonError(w, "Password must be at least 12 characters", http.StatusBadRequest)
			return
		}

		if !auth.ValidateTOTP(totpCode, totpSecret) {
			h.jsonError(w, "Invalid 2FA code", http.StatusBadRequest)
			return
		}

		hash, err := auth.HashPassword(password)
		if err != nil {
			h.jsonError(w, "Internal error", http.StatusInternalServerError)
			return
		}

		if err := h.DB.CreateUser(email, hash, totpSecret); err != nil {
			h.jsonError(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		_ = h.DB.SetTOTPVerified(1)

		h.jsonOK(w, map[string]bool{"success": true})
		return
	}

	h.jsonError(w, "Invalid step", http.StatusBadRequest)
}

// APILogin handles POST /api/v1/auth/login.
func (h *Handler) APILogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := clientIP(r)

	if !h.Limiter.Allow(ip) {
		h.jsonError(w, "Too many attempts. Please wait before trying again.", http.StatusTooManyRequests)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		TOTPCode string `json:"totp_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(req.Email)
	password := req.Password
	totpCode := req.TOTPCode

	if email == "" || password == "" || totpCode == "" {
		h.jsonError(w, "All fields are required", http.StatusBadRequest)
		return
	}

	user, err := h.DB.GetUserByEmail(email)
	if err != nil || user == nil {
		_ = h.DB.RecordLoginAttempt(ip, false)
		h.jsonError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	ok, err := auth.VerifyPassword(password, user.PasswordHash)
	if err != nil || !ok {
		_ = h.DB.RecordLoginAttempt(ip, false)
		h.jsonError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !auth.ValidateTOTP(totpCode, user.TOTPSecret) {
		_ = h.DB.RecordLoginAttempt(ip, false)
		h.jsonError(w, "Invalid 2FA code", http.StatusUnauthorized)
		return
	}

	token, err := h.JWT.Issue(user.ID, user.Email)
	if err != nil {
		h.jsonError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	_ = h.DB.RecordLoginAttempt(ip, true)
	h.Limiter.Reset(ip)

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(24 * time.Hour / time.Second),
	})

	h.jsonOK(w, map[string]bool{"success": true})
}

// APILogout handles POST /api/v1/auth/logout.
func (h *Handler) APILogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	h.jsonOK(w, map[string]bool{"success": true})
}

// generateQRDataURL creates a data:image/png;base64,... URL from an otpauth:// URL.
func generateQRDataURL(otpURL string) string {
	key, err := otp.NewKeyFromURL(otpURL)
	if err != nil {
		return ""
	}
	img, err := key.Image(256, 256)
	if err != nil {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("data:image/png;base64,")
	writer := base64.NewEncoder(base64.StdEncoding, &buf)
	if err := png.Encode(writer, img); err != nil {
		return ""
	}
	writer.Close()
	return buf.String()
}

// buildOTPURL recreates an otpauth:// URL from email + secret.
func buildOTPURL(email, secret string) string {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Postpilot",
		AccountName: email,
		Secret:      []byte(secret),
	})
	if err != nil {
		return ""
	}
	return key.URL()
}
