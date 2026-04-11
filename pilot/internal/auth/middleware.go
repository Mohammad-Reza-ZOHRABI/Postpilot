package auth

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey string

// UserContextKey is the key used to store Claims in the request context.
const UserContextKey contextKey = "user"

// RequireAuthJSON returns middleware that checks for a valid JWT in the "session" cookie.
// If invalid or missing, it returns a 401 JSON response instead of a redirect.
func RequireAuthJSON(jwtMgr *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
				return
			}

			_, err = jwtMgr.Validate(cookie.Value)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or expired session"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders adds standard security headers to all responses.
//
// CSP: script-src is strictly 'self'. The Svelte UI does NOT emit inline
// scripts. style-src keeps 'unsafe-inline' because Svelte/Vite output
// contains inline style attributes. img-src allows data: for the TOTP QR
// code which is embedded as a data URL.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; "+
				"connect-src 'self'; "+
				"font-src 'self'; "+
				"object-src 'none'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// rateLimitEntry tracks request attempts from a single IP.
type rateLimitEntry struct {
	count     int
	expiresAt time.Time
}

// RateLimiter provides simple in-memory per-IP rate limiting.
type RateLimiter struct {
	mu          sync.Mutex
	entries     map[string]*rateLimitEntry
	maxAttempts int
	window      time.Duration
}

// NewRateLimiter creates a RateLimiter that allows maxAttempts requests per IP
// within the given time window. It starts a background goroutine that cleans up
// expired entries every 5 minutes.
func NewRateLimiter(maxAttempts int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		entries:     make(map[string]*rateLimitEntry),
		maxAttempts: maxAttempts,
		window:      window,
	}

	go rl.cleanup()

	return rl
}

// Allow checks whether the given IP is allowed to make another request.
// Returns true if under the limit, false if the limit has been reached.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[ip]

	if !exists || now.After(entry.expiresAt) {
		rl.entries[ip] = &rateLimitEntry{
			count:     1,
			expiresAt: now.Add(rl.window),
		}
		return true
	}

	entry.count++
	return entry.count <= rl.maxAttempts
}

// Reset clears the rate limit counter for the given IP.
func (rl *RateLimiter) Reset(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.entries, ip)
}

// cleanup removes expired entries every 5 minutes.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, entry := range rl.entries {
			if now.After(entry.expiresAt) {
				delete(rl.entries, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// ---------------------------------------------------------------------------
// Per-API-key rate limiter (sliding 1-minute window).
// ---------------------------------------------------------------------------

// apiKeyBucket tracks request counts within the current minute for one key.
type apiKeyBucket struct {
	count      int
	windowEnds time.Time
}

// APIKeyRateLimiter enforces per-API-key request limits over a 1-minute
// sliding window. The limit is configured per-key in the database (see
// api_keys.rate_limit). This limiter is separate from the IP-based
// RateLimiter used for login attempts.
//
// Storage is in-memory — limits reset across process restarts. For a
// multi-process deployment, swap to a shared store (Redis) before scaling.
type APIKeyRateLimiter struct {
	mu      sync.Mutex
	entries map[int64]*apiKeyBucket
}

// NewAPIKeyRateLimiter creates an APIKeyRateLimiter and starts its cleanup
// goroutine.
func NewAPIKeyRateLimiter() *APIKeyRateLimiter {
	rl := &APIKeyRateLimiter{entries: make(map[int64]*apiKeyBucket)}
	go rl.cleanup()
	return rl
}

// Allow returns true if the given key is under its per-minute limit.
// A limit of zero or less is treated as no limit (always allowed).
func (rl *APIKeyRateLimiter) Allow(keyID int64, limit int) bool {
	if limit <= 0 {
		return true
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.entries[keyID]
	if !ok || now.After(b.windowEnds) {
		rl.entries[keyID] = &apiKeyBucket{count: 1, windowEnds: now.Add(time.Minute)}
		return true
	}
	b.count++
	return b.count <= limit
}

func (rl *APIKeyRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for id, b := range rl.entries {
			if now.After(b.windowEnds) {
				delete(rl.entries, id)
			}
		}
		rl.mu.Unlock()
	}
}

// ---------------------------------------------------------------------------
// Client IP extraction.
// ---------------------------------------------------------------------------

// ClientIP extracts the client IP address from the request.
// It checks X-Forwarded-For first, then falls back to RemoteAddr.
//
// SECURITY: X-Forwarded-For is trusted unconditionally. Postpilot MUST be
// deployed behind a trusted reverse proxy (Traefik, nginx, Caddy) configured
// to strip any client-supplied X-Forwarded-For and append only the real peer
// IP. If Postpilot is exposed directly to the internet, rate limiting can be
// bypassed by spoofing the header. See SECURITY.md → Deployment Checklist.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs; the first is the client
		if idx := strings.IndexByte(xff, ','); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// RemoteAddr is in the form "host:port"
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
