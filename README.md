# Postpilot

Hardened, minimal Docker image combining **Mailpit** (SMTP catcher + web UI) + **Postfix** (MTA) + **OpenDKIM** (DKIM signing).

- **Base**: Alpine 3.21 · **Size**: ~85 MB
- **Process supervision**: s6-overlay v3
- **Architectures**: `linux/amd64`, `linux/arm64`
- **DockerHub**: `rezozohrabi/postpilot`

---

## Modes

| `MAIL_MODE` | Behaviour |
|-------------|-----------|
| `catch` *(default)* | Mailpit stores & displays emails — **nothing is sent** |
| `relay` | Mailpit → Postfix → external SMTP relay (SendGrid, AWS SES…) |
| `direct` | Mailpit → Postfix → MX servers directly (requires PTR + DKIM) |

In all modes, **your apps always connect to port 1025** (Mailpit SMTP). The mode only affects whether Mailpit forwards to Postfix.

---

## Quick Start

### Development (catch mode)

```bash
docker run -d \
  --name mailserver \
  -p 127.0.0.1:1025:1025 \
  -p 127.0.0.1:8025:8025 \
  rezozohrabi/postpilot:latest
```

Open `http://localhost:8025` to see captured emails.

### Production (relay mode)

```bash
mkdir -p .secrets
echo -n "your_smtp_password" > .secrets/smtp_relay_password

docker run -d \
  --name mailserver \
  -e MAIL_MODE=relay \
  -e POSTFIX_HOSTNAME=mail.example.com \
  -e POSTFIX_MYORIGIN=example.com \
  -e POSTFIX_RELAY_HOST='[smtp.sendgrid.net]:587' \
  -e POSTFIX_RELAY_USER=apikey \
  --secret source=smtp_relay_password,target=/run/secrets/smtp_relay_password \
  -p 127.0.0.1:1025:1025 \
  -p 127.0.0.1:8025:8025 \
  rezozohrabi/postpilot:latest
```

### Production (direct mode)

```bash
docker run -d \
  --name mailserver \
  -e MAIL_MODE=direct \
  -e POSTFIX_HOSTNAME=mail.example.com \
  -e POSTFIX_MYORIGIN=example.com \
  -e DKIM_ENABLED=true \
  -e DKIM_DOMAIN=example.com \
  -p 127.0.0.1:1025:1025 \
  -p 127.0.0.1:8025:8025 \
  rezozohrabi/postpilot:latest
```

Check logs for the auto-generated DKIM DNS record:
```bash
docker logs mailserver 2>&1 | grep -A5 "DNS TXT"
```

---

## Docker Compose

```bash
cp docker-compose.example.yml docker-compose.yml
cp .env.example .env
mkdir -p .secrets
# Fill in .env and .secrets/ files
docker compose up -d
```

### With Traefik (HTTPS + BasicAuth)

```bash
# Generate auth hash (escape $ as $$ for Docker Compose)
htpasswd -nbB -C 12 admin yourpassword | sed 's/\$/\$\$/g'
# Add the output to .env as TRAEFIK_AUTH_USERS=...

docker compose -f docker-compose.yml -f docker-compose.traefik.yml up -d
```

---

## Environment Variables

### Core

| Variable | Default | Description |
|----------|---------|-------------|
| `MAIL_MODE` | `catch` | `catch` / `relay` / `direct` |
| `POSTFIX_HOSTNAME` | `mail.local` | Postfix `myhostname` (FQDN) |
| `POSTFIX_MYORIGIN` | `mail.local` | Postfix `myorigin` (domain) |
| `POSTFIX_MESSAGE_SIZE` | `52428800` | Max message size in bytes (50 MB) |
| `POSTFIX_MYNETWORKS` | `127.0.0.0/8` | Allowed sender networks |

### Relay mode

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTFIX_RELAY_HOST` | — | Relay host, e.g. `[smtp.sendgrid.net]:587` |
| `POSTFIX_RELAY_USER` | — | SMTP auth username |

### DKIM

| Variable | Default | Description |
|----------|---------|-------------|
| `DKIM_ENABLED` | `false` | Enable OpenDKIM signing |
| `DKIM_DOMAIN` | `POSTFIX_MYORIGIN` | Domain for DKIM key |
| `DKIM_SELECTOR` | `default` | DKIM selector |
| `DKIM_KEY_SIZE` | `2048` | Key size for auto-generation (`2048` or `4096`) |

### TLS & Mailpit

| Variable | Default | Description |
|----------|---------|-------------|
| `TLS_ENABLED` | `false` | Enable TLS on Postfix |
| `MP_SMTP_PORT` | `1025` | Mailpit SMTP port |
| `MP_UI_PORT` | `8025` | Mailpit web UI port |
| `MP_MAX_MESSAGES` | `500` | Max stored messages |

### Traefik overlay

| Variable | Default | Description |
|----------|---------|-------------|
| `MAILPIT_DOMAIN` | **required** | Domain for web UI |
| `TRAEFIK_AUTH_USERS` | **required** | `htpasswd` bcrypt hash (`$$` escaping) |
| `TRAEFIK_NETWORK` | `traefik_network` | External Traefik network name |
| `TRAEFIK_ENTRYPOINT` | `websecure` | Traefik HTTPS entrypoint |
| `TRAEFIK_CERTRESOLVER` | `letsencrypt` | Certificate resolver |

### Container resources

| Variable | Default | Description |
|----------|---------|-------------|
| `MAILSERVER_IMAGE` | `rezozohrabi/postpilot:latest` | Docker image |
| `MAILSERVER_CONTAINER_NAME` | `mailserver` | Container name |
| `MAILSERVER_MEMORY_LIMIT` | `128M` | Memory limit |
| `MAILSERVER_CPU_LIMIT` | `0.25` | CPU limit |

---

## Docker Secrets

| Secret | Content | Description |
|--------|---------|-------------|
| `smtp_relay_password` | Plain text | SMTP relay password |
| `dkim_private_key` | Base64-encoded PEM | DKIM private key |
| `tls_cert` | Base64-encoded PEM | TLS certificate |
| `tls_key` | Base64-encoded PEM | TLS private key |
| `mailpit_auth` | `user:$2y$hash` | Mailpit UI basic auth (bcrypt) |

```bash
# Generate bcrypt hash for Mailpit auth
htpasswd -bnBC 12 admin yourpassword > .secrets/mailpit_auth
```

---

## DKIM Setup

If no DKIM key is provided, one is **auto-generated** at startup.

```bash
# Option 1: Auto-generate (key is logged on first start)
docker run ... -e DKIM_ENABLED=true -e DKIM_DOMAIN=example.com ...
docker logs mailserver 2>&1 | grep -A5 "DNS TXT"

# Option 2: Provide your own key
openssl genrsa -out dkim.pem 2048
base64 -w0 dkim.pem > .secrets/dkim_private_key
```

Add the printed DNS TXT record to your domain:
```
default._domainkey.example.com  TXT  "v=DKIM1; k=rsa; p=..."
```

### DNS records for direct mode

| Type | Name | Value |
|------|------|-------|
| A | `mail` | `<your-vps-ip>` |
| MX | `@` | `mail.example.com` (priority 10) |
| TXT | `@` | `v=spf1 ip4:<your-vps-ip> -all` |
| TXT | `default._domainkey` | `v=DKIM1; k=rsa; p=...` |
| TXT | `_dmarc` | `v=DMARC1; p=reject; adkim=s; aspf=s; rua=mailto:dmarc@example.com` |
| PTR | `<your-vps-ip>` | `mail.example.com` (configure at VPS provider) |

---

## Hardening

This image is designed to run with minimal privileges:

```yaml
security_opt:
  - no-new-privileges:true
cap_drop: [ALL]
cap_add: [SETUID, SETGID, CHOWN, DAC_OVERRIDE]
tmpfs:
  - /tmp:mode=1777
```

- **Mailpit** runs as `mailpit` (UID 10001)
- **Postfix** uses its own internal system users (drops privileges by design)
- **OpenDKIM** runs as `opendkim`
- **Seccomp** enabled (Docker default profile)
- Secrets via `/run/secrets/` files — never in environment variables
- TLS outbound: `>=TLSv1.2`, ciphers `high`

---

## Integration with your app

```env
# In your application's .env or docker-compose.yml
SMTP_HOST=mailserver
SMTP_PORT=1025
```

The `mailserver` container and your app must share the same Docker network.

---

## Build from source

```bash
git clone https://github.com/Mohammad-Reza-ZOHRABI/Postpilot
cd Mail-Server
docker build -t mailpit-postfix:local .
```

---

## License

MIT
