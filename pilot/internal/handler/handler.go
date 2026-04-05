package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/auth"
	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/database"
	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/mail"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	DB      *database.DB
	JWT     *auth.JWTManager
	Mail    *mail.Client
	Limiter *auth.RateLimiter
}

// New creates a Handler.
func New(db *database.DB, jwt *auth.JWTManager, mailer *mail.Client, limiter *auth.RateLimiter) *Handler {
	return &Handler{
		DB:      db,
		JWT:     jwt,
		Mail:    mailer,
		Limiter: limiter,
	}
}

// jsonOK writes a 200 JSON response.
func (h *Handler) jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// jsonError writes an error JSON response with the given status code.
func (h *Handler) jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// authenticateFromCookie reads the JWT from the "session" cookie and returns
// the parsed claims. Returns nil if the cookie is absent or the token is invalid.
func (h *Handler) authenticateFromCookie(r *http.Request) *auth.Claims {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}
	claims, err := h.JWT.Validate(cookie.Value)
	if err != nil {
		return nil
	}
	return claims
}

// clientIP extracts the client IP from the request.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
