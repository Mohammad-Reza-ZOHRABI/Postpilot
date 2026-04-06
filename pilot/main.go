package main

import (
	"crypto/rand"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/auth"
	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/database"
	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/handler"
	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/internal/mail"
	"github.com/Mohammad-Reza-ZOHRABI/Postpilot/pilot/web"
)

func main() {
	// Config from environment
	dbPath := env("PILOT_DB_PATH", "/data/postpilot/pilot.db")
	listenAddr := ":" + env("PILOT_PORT", "3000")
	smtpHost := env("PILOT_SMTP_HOST", "127.0.0.1")
	smtpPort := env("PILOT_SMTP_PORT", "1025")

	// Ensure data directory
	if err := os.MkdirAll("/data/postpilot", 0755); err != nil {
		log.Fatal("Failed to create data dir:", err)
	}

	// Open database
	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatal("Database:", err)
	}
	defer db.Close()

	// JWT secret — generate if not stored
	jwtSecret, err := getOrCreateJWTSecret(db)
	if err != nil {
		log.Fatal("JWT secret:", err)
	}
	jwtMgr := auth.NewJWTManager(jwtSecret, 24*time.Hour)

	// Services
	mailer := mail.NewClient(smtpHost, smtpPort)
	limiter := auth.NewRateLimiter(5, 15*time.Minute)

	h := handler.New(db, jwtMgr, mailer, limiter)

	// Auth middleware for cookie-protected routes
	requireAuth := auth.RequireAuthJSON(jwtMgr)

	// Router
	mux := http.NewServeMux()

	// ---------- Public API routes ----------
	mux.HandleFunc("/api/v1/health", h.APIHealth)
	mux.HandleFunc("/api/v1/auth/check", h.APIAuthCheck)
	mux.HandleFunc("/api/v1/auth/setup", h.APISetup)
	mux.HandleFunc("/api/v1/auth/login", h.APILogin)
	mux.HandleFunc("/api/v1/auth/logout", h.APILogout)

	// ---------- API-key authenticated routes ----------
	mux.HandleFunc("/api/v1/send", h.APISend)
	mux.HandleFunc("/api/v1/status/", h.APIStatus)

	// ---------- Cookie-authenticated routes ----------
	mux.Handle("/api/v1/dashboard", requireAuth(http.HandlerFunc(h.APIDashboard)))
	mux.Handle("/api/v1/settings", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.APIGetSettings(w, r)
		case http.MethodPost:
			h.APISaveSettings(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, `{"error":"Method not allowed"}`)
		}
	})))
	mux.Handle("/api/v1/keys", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.APIGetKeys(w, r)
		case http.MethodPost:
			h.APICreateKey(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, `{"error":"Method not allowed"}`)
		}
	})))
	mux.Handle("/api/v1/keys/", requireAuth(http.HandlerFunc(h.APIRevokeKey)))
	mux.Handle("/api/v1/dns/check", requireAuth(http.HandlerFunc(h.APIDNSCheck)))

	// ---------- User management routes ----------
	mux.Handle("/api/v1/users", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.APIListUsers(w, r)
		case http.MethodPost:
			h.APICreateUser(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, `{"error":"Method not allowed"}`)
		}
	})))
	mux.Handle("/api/v1/users/", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/delete") {
			h.APIDeleteUser(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/role") {
			h.APIUpdateUserRole(w, r)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":"Not found"}`)
		}
	})))

	// ---------- SPA handler ----------
	mux.Handle("/", spaHandler())

	// Wrap everything with security headers
	var root http.Handler = mux
	root = auth.SecurityHeaders(root)

	// Start server
	log.Printf("[pilot] Postpilot API starting on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, root); err != nil {
		log.Fatal("Server:", err)
	}
}

// spaHandler returns an http.Handler that serves the embedded Svelte SPA.
// It serves files from the embedded dist/ directory. If a file is not found,
// it falls back to serving index.html for client-side routing.
func spaHandler() http.Handler {
	distFS, err := fs.Sub(web.UI, "dist")
	if err != nil {
		log.Fatal("Failed to create sub FS for dist:", err)
	}
	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the path
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		// Try to open the file in the embedded FS
		f, err := distFS.Open(path)
		if err != nil {
			// File not found — serve index.html for SPA client-side routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()

		// File exists — serve it normally
		fileServer.ServeHTTP(w, r)
	})
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getOrCreateJWTSecret(db *database.DB) ([]byte, error) {
	secret, err := db.GetSetting("jwt_secret")
	if err == nil && secret != "" {
		return []byte(secret), nil
	}

	// Generate new 256-bit secret
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}
	secretStr := fmt.Sprintf("%x", key)
	if err := db.SetSetting("jwt_secret", secretStr); err != nil {
		return nil, err
	}
	return []byte(secretStr), nil
}
