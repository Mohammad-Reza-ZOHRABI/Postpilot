# Security Policy

Postpilot is an open-source self-hosted email infrastructure built from
Mailpit + Postfix + OpenDKIM with a Go admin UI (Pilot) and a Svelte frontend.
It is deployed in production at `mail.re-zo.tech`.

Last security audit: **2026-04-10** (pragmatic audit v1).

---

## Supported Versions

Postpilot is currently pre-1.0. Only the latest commit on the `main` branch
receives security updates.

| Version | Supported |
|---------|-----------|
| `main` (latest) | Yes |
| older tags / branches | No |

---

## Reporting a Vulnerability

Please **do not open a public GitHub issue** for security vulnerabilities.

### Preferred: GitHub Security Advisories

Use the private advisory workflow built into GitHub:

https://github.com/Mohammad-Reza-ZOHRABI/Postpilot/security/advisories/new

### Alternate: email

`security@re-zo.tech`

### What to expect

- **First response**: within 72 hours
- **Triage & fix window**: critical issues targeted within 7 days, high within 30 days
- **Public disclosure**: we ask for a 90-day responsible disclosure window from
  first report; if a fix is available sooner, we coordinate disclosure with you
- **Credit**: reporters are credited in the security changelog unless they
  prefer to remain anonymous
- **PGP**: optional; key fingerprint available on request

### Scope

In scope:
- The Pilot Go backend (`pilot/`) and Svelte UI (`pilot/ui/`)
- The Mailpit / Postfix / OpenDKIM container image built from this repository
- The bundled `configure-all.sh` and s6-overlay service scripts
- HTTP API endpoints (`/api/v1/*`)
- Docker Compose configurations shipped with this repository

Out of scope:
- DNS configuration errors on the operator's side (SPF, DKIM, DMARC, PTR)
- The upstream Postfix, OpenDKIM, or Mailpit projects — report those upstream
- Self-hosted deployments not following the [Deployment Security Checklist](#deployment-security-checklist)
  (notably: exposing Postpilot directly to the internet without a trusted
  reverse proxy, which breaks rate limiting assumptions)

---

## Security Measures

### Authentication & Session

- **Password hashing**: Argon2id with parameters `time=1`, `memory=64 MiB`,
  `threads=4`, `keyLen=32`, `saltLen=16`. Hash format:
  `$argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>`. Verification uses
  `crypto/subtle.ConstantTimeCompare` (timing-safe). Source:
  `pilot/internal/auth/password.go`.
- **Mandatory TOTP 2FA**: RFC 6238 via `github.com/pquerna/otp`. Every user
  must configure TOTP at setup or first login. Source:
  `pilot/internal/auth/totp.go`.
- **JWT sessions**: HS256, 24-hour expiry. Secret is 32 random bytes generated
  on first boot and persisted in the SQLite `settings` table (never in env
  vars, never logged). Source: `pilot/internal/auth/jwt.go`, `main.go`.
- **Session cookies**: `HttpOnly`, `Secure` (when TLS is terminated by the
  app), `SameSite=Strict`, scoped to `/`. Set in `api_auth.go`.
- **Login rate limiting**: 5 attempts per 15 minutes per IP (in-memory, sliding
  window). Applied to `POST /api/v1/auth/login`.
- **Login audit logging**: success/failure per IP logged to the
  `login_attempts` table.

### Authorization

- **Role-based access control**: two roles — `admin` and `member`.
  Every admin-only endpoint checks `claims.Role != "admin"` explicitly and
  returns `403 Forbidden`.
- **First user bootstrap**: the `/api/v1/auth/setup` route is only reachable
  when the database has zero users. Once set up, it is unreachable.
- **API key permissions**: each key has a `permissions` field (e.g. `send`,
  `status`). Keys are hashed with SHA-256 before storage and only the first 16
  characters of the plaintext key are ever returned (once, at creation).
- **Revoked keys**: rejected on validation via a `revoked_at` timestamp.

### Input Validation

- **Email addresses**: validated with `net/mail.ParseAddress` on `From`,
  `To`, and `Reply-To` in `/api/v1/send`.
- **Password minimum length**: 12 characters, enforced at setup and user
  creation.
- **Numeric IDs**: hand-rolled digit-only parsing for `/api/v1/status/:id`.
- **JSON decoding**: strict decoding via `json.NewDecoder(r.Body).Decode`.

### Secrets Management

- **Docker secrets** preferred over environment variables for:
  `smtp_relay_password`, `dkim_private_key`, `tls_cert`, `tls_key`,
  `mailpit_auth`. See `rootfs/etc/s6-overlay/scripts/configure-all.sh`.
- **API keys**: 32 random bytes generated via `crypto/rand`, SHA-256-hashed,
  only the first 16 chars of the plaintext are retained (for identification in
  lists). Keys are displayed exactly once, at creation.
- **JWT secret**: 32 random bytes, stored in the DB `settings` table. Never
  logged, never written to disk outside the SQLite file.
- **DKIM private key**: 2048-bit RSA (or 4096-bit, configurable). Auto-
  generated at first boot if not provided via a Docker secret.
- **Secret files**: `.secrets/` directory is gitignored. File permissions
  `0600` at rest.

### HTTP & Transport Security

- **Strict-Transport-Security**: `max-age=31536000; includeSubDomains`
- **Content-Security-Policy**:
  ```
  default-src 'self';
  script-src 'self';                    (no 'unsafe-inline')
  style-src 'self' 'unsafe-inline';     (required by Svelte/Vite inline styles)
  img-src 'self' data:;                 (required for TOTP QR codes)
  connect-src 'self';
  font-src 'self';
  object-src 'none';
  frame-ancestors 'none';
  base-uri 'self';
  form-action 'self';
  ```
- **X-Frame-Options**: `DENY`
- **X-Content-Type-Options**: `nosniff`
- **Referrer-Policy**: `strict-origin-when-cross-origin`
- **Permissions-Policy**: `camera=(), microphone=(), geolocation=()`
- **CSRF**: `SameSite=Strict` cookies block cross-site requests; no additional
  CSRF tokens needed.

### Container Hardening

- **Alpine 3.21 base** (~5 MB) with minimal packages.
- **Multi-stage build**: s6-overlay, Mailpit, Svelte UI, Go binary (static,
  `CGO_ENABLED=0`, `-trimpath`, `-ldflags="-s -w"`), production image.
- **Non-root runtime users**:
  - `mailpit` (UID 10001) runs Mailpit
  - `pilot` (UID 10002) runs the Pilot backend
  - `opendkim` runs OpenDKIM (system user)
  - `postfix` runs Postfix (system user)
- **Capabilities**: `cap_drop: ALL` + only `SETUID`, `SETGID`, `CHOWN`,
  `DAC_OVERRIDE` added.
- **security_opt**: `no-new-privileges:true`
- **tmpfs** on `/tmp`
- **Postfix listens only on loopback** (`inet_interfaces = loopback-only`,
  port 2525)
- **OpenDKIM oversigns** the From header.

### Audit & Observability

- `login_attempts` table: every login success/failure with IP and timestamp
- `email_logs` table: every send with status, from, to, subject, error
- API key `call_count` and `last_used_at` counters incremented on every use
- Pilot errors logged via `log.Printf` to the container stdout; Docker
  captures them for `docker logs`

### Rate Limiting

- **Login**: 5 attempts / 15 minutes per IP
- **Per-API-key rate limit**: enforced per-minute, configured in the
  `api_keys.rate_limit` column. Requests beyond the limit return `429
  Too Many Requests` with `Retry-After: 60`.

### Outbound Email Safety

- **Defense-in-depth HTML sanitization** on bodies submitted via
  `/api/v1/send`: strips `<script>`, `<iframe>`, `<object>`, `<embed>`,
  `<applet>`, inline `on*=` event handlers, and neutralizes `javascript:`,
  `vbscript:`, `data:text/html` URL schemes in href/src/action attributes.
  Source: `pilot/internal/mail/sanitize.go`. This is a best-effort regex-based
  sanitizer, not a full HTML parser — see [Known Limitations](#known-limitations).
- **Header injection prevention**: Postfix enforces RFC 821 envelopes and
  strips CRLF in headers.

### Dependency Management

- **Dependabot** enabled on `gomod` (backend), `npm` (Svelte UI), `docker`,
  and `github-actions` — weekly security updates.
- **govulncheck** runs in the Dockerfile `pilot-build` stage and blocks the
  build if a known vulnerability is found in any Go dependency.

---

## Known Limitations

These are accepted trade-offs documented for transparency. Report any if you
find an exploit, but they are not treated as vulnerabilities on their own.

- **Rate limiters are in-memory.** Per-IP and per-API-key limits are tracked
  in process memory and reset on restart. Acceptable for a single-node MVP;
  move to Redis when scaling to multiple Pilot instances.
- **No refresh tokens.** JWTs are 24-hour one-shot. Users must re-authenticate
  daily. Acceptable trade-off for an admin tool with 2FA.
- **No account lockout beyond per-IP login rate limiting.** A distributed
  brute-force spread across many IPs would not trip any lockout. Mitigated
  by mandatory TOTP 2FA and Argon2id hashing.
- **`X-Forwarded-For` is trusted unconditionally.** Postpilot MUST be deployed
  behind a trusted reverse proxy (Traefik, nginx, Caddy) configured to strip
  any client-supplied `X-Forwarded-For` and append only the real peer IP. If
  Postpilot is exposed directly to the internet, rate limiting can be
  bypassed by spoofing the header. See
  `pilot/internal/auth/middleware.go` `ClientIP` and the deployment checklist
  below.
- **HTML sanitization is regex-based** (`pilot/internal/mail/sanitize.go`),
  not a full HTML parser. It strips the most common dangerous constructs but
  can be defeated by crafted input. It is a defense-in-depth layer — do not
  rely on it as the sole XSS defense if you later render email bodies in a
  user-facing UI.
- **SQLite is a single file.** Back up by snapshotting the Docker volume;
  there is no built-in hot backup.
- **CSP allows `'unsafe-inline'` for styles.** Svelte/Vite output contains
  inline style attributes; removing this would require a nonce-based CSP
  build step.
- **No OAuth2 / SSO.** The admin UI uses local accounts only.

---

## Security Changelog

### 2026-04-10 — Pragmatic security audit v1

Fixed:

- **CRITICAL** — `api_keys.rate_limit` was stored in the database but never
  enforced in the request path. Added `APIKeyRateLimiter` (per-key,
  sliding 1-minute window) and wired it into `/api/v1/send` and
  `/api/v1/status/*`. Exceeded requests now return `429` with `Retry-After: 60`.
- **CRITICAL** — Handler error responses concatenated raw `err.Error()` into
  the JSON body (e.g., `"Failed to save key: UNIQUE constraint failed:
  api_keys.key_hash"`). This leaked SQL schema and driver details. All
  handlers (`api.go`, `api_auth.go`, `api_keys.go`, `api_users.go`,
  `api_settings.go`) now return generic messages and log the real error
  server-side via `log.Printf`.
- **HIGH** — `/api/v1/send` did not validate email addresses. Added
  `net/mail.ParseAddress` checks on `From`, all `To` entries, and `Reply-To`.
- **HIGH** — No HTML sanitization of email bodies submitted via the API.
  Added `pilot/internal/mail/sanitize.go` (regex-based defense-in-depth
  sanitizer) called before `mail.Send`.
- **HIGH** — CSP `script-src` included `'unsafe-inline'`. Tightened to
  `script-src 'self'` only. The Svelte build does not emit inline scripts,
  so this change has no functional impact.
- **DOC** — Documented the `X-Forwarded-For` trust assumption in
  `middleware.go::ClientIP` with a reference to this document.

Added:

- `SECURITY.md` (this document)
- `.github/dependabot.yml` (weekly gomod + npm, monthly docker +
  github-actions)
- `govulncheck` in the Dockerfile `pilot-build` stage (blocks build on Go
  CVEs)

---

## Deployment Security Checklist

Administrators deploying Postpilot must follow these steps. Skipping any of
them may introduce vulnerabilities outside the project's control.

### Network & TLS

- [ ] **Deploy behind a trusted reverse proxy** (Traefik, nginx, or Caddy)
      that:
      - Terminates TLS
      - **Strips any client-supplied `X-Forwarded-For` header** and appends
        only the real peer IP (see the Traefik overlay at
        `docker-compose.traefik.yml` for a working example)
- [ ] **Do not expose port 25 (Postfix SMTP) publicly.** Postfix is
      configured to listen only on the Docker internal network.
- [ ] **Do not expose the Pilot admin UI (port 3000) without the reverse
      proxy.** The proxy enforces HTTPS and rate limiting at the edge.

### DNS records (required for `MAIL_MODE=direct`)

- [ ] **A record** for `mail.yourdomain.com` → your VPS IP
- [ ] **MX record** for `yourdomain.com` → `mail.yourdomain.com` priority 10
- [ ] **SPF TXT** for `yourdomain.com` → `v=spf1 ip4:<VPS_IP> -all`
- [ ] **DKIM TXT** for `default._domainkey.yourdomain.com` → value printed
      by Pilot on first start (check `docker logs mailserver`)
- [ ] **DMARC TXT** for `_dmarc.yourdomain.com` → `v=DMARC1; p=reject;
      adkim=s; aspf=s; rua=mailto:dmarc@yourdomain.com`
- [ ] **PTR record** (reverse DNS) for your VPS IP → `mail.yourdomain.com`
      (configured with your VPS provider)

### Secrets & authentication

- [ ] Create the initial admin account via `/setup` on first run. The setup
      route becomes unreachable after the first user is created.
- [ ] **Enroll TOTP immediately** and store the recovery secret in a password
      manager.
- [ ] Use Docker secrets for `smtp_relay_password`, `dkim_private_key`,
      `tls_cert`, `tls_key`, `mailpit_auth` — not environment variables.
- [ ] **Rotate API keys periodically** from the admin UI. Revoked keys
      continue to exist in the database for audit but cannot be used.
- [ ] Set a per-key `rate_limit` appropriate for the consuming application
      (default: 100 requests/minute).

### Persistence & backups

- [ ] Mount `/data/postpilot` (Pilot SQLite DB) and `/data/mailpit` (Mailpit
      storage) on a persistent Docker volume.
- [ ] Back up both volumes regularly. SQLite backups must either stop the
      container briefly or use the `.backup` SQLite command.
- [ ] Snapshot the volumes before any container upgrade.

### Monitoring

- [ ] **Enable Dependabot** in GitHub (Settings → Code security and analysis →
      Dependabot alerts + security updates).
- [ ] Monitor `docker logs mailserver` for repeated rate-limit violations or
      failed login attempts.
- [ ] Review the `login_attempts` table periodically.
