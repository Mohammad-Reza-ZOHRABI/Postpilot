package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// ---------- Types ----------

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	TOTPSecret   string
	TOTPVerified bool
	CreatedAt    time.Time
}

type APIKey struct {
	ID          int64
	Name        string
	KeyHash     string
	KeyPrefix   string
	Permissions string
	RateLimit   int
	LastUsedAt  *time.Time
	CallCount   int64
	CreatedAt   time.Time
	RevokedAt   *time.Time
}

type EmailLog struct {
	ID             int64
	APIKeyID       *int64
	FromAddr       string
	ToAddr         string
	Subject        string
	Status         string
	PostfixQueueID string
	ErrorMsg       string
	CreatedAt      time.Time
}

type EmailStats struct {
	Sent24h   int
	Sent7d    int
	Sent30d   int
	Failed24h int
	Queued    int
}

// ---------- DB handle ----------

type DB struct {
	db *sql.DB
}

// Open creates or opens the SQLite database at path, runs pragmas and migrations.
func Open(path string) (*DB, error) {
	if path == "" {
		path = "/data/postpilot/pilot.db"
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("database: mkdir: %w", err)
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}

	// Single connection avoids locking issues with SQLite.
	conn.SetMaxOpenConns(1)

	// Pragmas: WAL mode for concurrent reads, busy timeout 5 s, foreign keys on.
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := conn.Exec(pragma); err != nil {
			conn.Close()
			return nil, fmt.Errorf("database: pragma %q: %w", pragma, err)
		}
	}

	d := &DB{db: conn}
	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return d, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// ---------- Migrations ----------

func (d *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			totp_secret TEXT NOT NULL,
			totp_verified BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			key_hash TEXT UNIQUE NOT NULL,
			key_prefix TEXT NOT NULL,
			permissions TEXT DEFAULT 'send',
			rate_limit INTEGER DEFAULT 100,
			last_used_at DATETIME,
			call_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			revoked_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS email_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_key_id INTEGER,
			from_addr TEXT NOT NULL,
			to_addr TEXT NOT NULL,
			subject TEXT,
			status TEXT DEFAULT 'queued',
			postfix_queue_id TEXT,
			error_msg TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS login_attempts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT NOT NULL,
			success BOOLEAN NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, s := range stmts {
		if _, err := d.db.Exec(s); err != nil {
			return fmt.Errorf("database: migrate: %w", err)
		}
	}
	return nil
}

// ========== Users ==========

// HasUsers returns true if at least one user exists.
func (d *DB) HasUsers() (bool, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count > 0, err
}

// CreateUser inserts a new user row.
func (d *DB) CreateUser(email, passwordHash, totpSecret string) error {
	_, err := d.db.Exec(
		"INSERT INTO users (email, password_hash, totp_secret) VALUES (?, ?, ?)",
		email, passwordHash, totpSecret,
	)
	return err
}

// GetUserByEmail returns the user or nil if not found.
func (d *DB) GetUserByEmail(email string) (*User, error) {
	u := &User{}
	err := d.db.QueryRow(
		"SELECT id, email, password_hash, totp_secret, totp_verified, created_at FROM users WHERE email = ?",
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.TOTPSecret, &u.TOTPVerified, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// SetTOTPVerified marks the user's TOTP as verified.
func (d *DB) SetTOTPVerified(userID int64) error {
	_, err := d.db.Exec("UPDATE users SET totp_verified = 1 WHERE id = ?", userID)
	return err
}

// ========== API Keys ==========

// CreateAPIKey inserts a new API key row.
func (d *DB) CreateAPIKey(name, keyHash, keyPrefix, permissions string, rateLimit int) error {
	_, err := d.db.Exec(
		"INSERT INTO api_keys (name, key_hash, key_prefix, permissions, rate_limit) VALUES (?, ?, ?, ?, ?)",
		name, keyHash, keyPrefix, permissions, rateLimit,
	)
	return err
}

// GetAPIKeyByHash returns the API key matching the hash, or nil.
func (d *DB) GetAPIKeyByHash(hash string) (*APIKey, error) {
	k := &APIKey{}
	err := d.db.QueryRow(
		`SELECT id, name, key_hash, key_prefix, permissions, rate_limit,
		        last_used_at, call_count, created_at, revoked_at
		 FROM api_keys WHERE key_hash = ?`, hash,
	).Scan(&k.ID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Permissions, &k.RateLimit,
		&k.LastUsedAt, &k.CallCount, &k.CreatedAt, &k.RevokedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return k, err
}

// ListAPIKeys returns all API keys ordered by creation date descending.
func (d *DB) ListAPIKeys() ([]APIKey, error) {
	rows, err := d.db.Query(
		`SELECT id, name, key_hash, key_prefix, permissions, rate_limit,
		        last_used_at, call_count, created_at, revoked_at
		 FROM api_keys ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Permissions, &k.RateLimit,
			&k.LastUsedAt, &k.CallCount, &k.CreatedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// RevokeAPIKey sets the revoked_at timestamp for the given key.
func (d *DB) RevokeAPIKey(id int64) error {
	_, err := d.db.Exec(
		"UPDATE api_keys SET revoked_at = CURRENT_TIMESTAMP WHERE id = ?", id,
	)
	return err
}

// IncrementAPIKeyUsage bumps call_count and sets last_used_at.
func (d *DB) IncrementAPIKeyUsage(id int64) error {
	_, err := d.db.Exec(
		"UPDATE api_keys SET call_count = call_count + 1, last_used_at = CURRENT_TIMESTAMP WHERE id = ?", id,
	)
	return err
}

// ========== Settings ==========

// GetSetting returns the value for key, or empty string if not found.
func (d *DB) GetSetting(key string) (string, error) {
	var val string
	err := d.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

// SetSetting upserts a setting.
func (d *DB) SetSetting(key, value string) error {
	_, err := d.db.Exec(
		`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
		key, value,
	)
	return err
}

// GetAllSettings returns every setting as a map.
func (d *DB) GetAllSettings() (map[string]string, error) {
	rows, err := d.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, rows.Err()
}

// ========== Email Logs ==========

// LogEmail inserts an email log row and returns its id.
func (d *DB) LogEmail(apiKeyID *int64, from, to, subject, status string) (int64, error) {
	res, err := d.db.Exec(
		`INSERT INTO email_logs (api_key_id, from_addr, to_addr, subject, status)
		 VALUES (?, ?, ?, ?, ?)`,
		apiKeyID, from, to, subject, status,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateEmailStatus updates status, queue id, and error message for a log entry.
func (d *DB) UpdateEmailStatus(id int64, status, queueID, errMsg string) error {
	_, err := d.db.Exec(
		"UPDATE email_logs SET status = ?, postfix_queue_id = ?, error_msg = ? WHERE id = ?",
		status, queueID, errMsg, id,
	)
	return err
}

// GetEmailLog returns a single email log by id, or nil.
func (d *DB) GetEmailLog(id int64) (*EmailLog, error) {
	e := &EmailLog{}
	var queueID, errMsg sql.NullString
	err := d.db.QueryRow(
		`SELECT id, api_key_id, from_addr, to_addr, subject, status,
		        postfix_queue_id, error_msg, created_at
		 FROM email_logs WHERE id = ?`, id,
	).Scan(&e.ID, &e.APIKeyID, &e.FromAddr, &e.ToAddr, &e.Subject, &e.Status,
		&queueID, &errMsg, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if queueID.Valid {
		e.PostfixQueueID = queueID.String
	}
	if errMsg.Valid {
		e.ErrorMsg = errMsg.String
	}
	return e, nil
}

// GetEmailStats returns aggregate send/fail/queued counts.
func (d *DB) GetEmailStats() (*EmailStats, error) {
	s := &EmailStats{}
	err := d.db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN status = 'sent' AND created_at >= datetime('now', '-1 day')  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'sent' AND created_at >= datetime('now', '-7 days') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'sent' AND created_at >= datetime('now', '-30 days') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'failed' AND created_at >= datetime('now', '-1 day') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'queued' THEN 1 ELSE 0 END), 0)
		FROM email_logs
	`).Scan(&s.Sent24h, &s.Sent7d, &s.Sent30d, &s.Failed24h, &s.Queued)
	return s, err
}

// RecentEmails returns the most recent email logs up to limit.
func (d *DB) RecentEmails(limit int) ([]EmailLog, error) {
	rows, err := d.db.Query(
		`SELECT id, api_key_id, from_addr, to_addr, subject, status,
		        postfix_queue_id, error_msg, created_at
		 FROM email_logs ORDER BY created_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []EmailLog
	for rows.Next() {
		var e EmailLog
		var queueID, errMsg sql.NullString
		if err := rows.Scan(&e.ID, &e.APIKeyID, &e.FromAddr, &e.ToAddr, &e.Subject, &e.Status,
			&queueID, &errMsg, &e.CreatedAt); err != nil {
			return nil, err
		}
		if queueID.Valid {
			e.PostfixQueueID = queueID.String
		}
		if errMsg.Valid {
			e.ErrorMsg = errMsg.String
		}
		logs = append(logs, e)
	}
	return logs, rows.Err()
}

// ========== Login Attempts ==========

// RecordLoginAttempt logs a login attempt from the given IP.
func (d *DB) RecordLoginAttempt(ip string, success bool) error {
	_, err := d.db.Exec(
		"INSERT INTO login_attempts (ip, success) VALUES (?, ?)", ip, success,
	)
	return err
}

// RecentFailedAttempts counts failed login attempts from ip in the last N minutes.
func (d *DB) RecentFailedAttempts(ip string, minutes int) (int, error) {
	var count int
	err := d.db.QueryRow(
		`SELECT COUNT(*) FROM login_attempts
		 WHERE ip = ? AND success = 0 AND created_at >= datetime('now', '-' || ? || ' minutes')`,
		ip, minutes,
	).Scan(&count)
	return count, err
}
