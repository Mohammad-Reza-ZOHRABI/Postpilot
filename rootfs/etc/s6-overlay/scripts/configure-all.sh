#!/command/with-contenv sh
# =============================================================================
# configure-all.sh — Central configuration generator
# Runs once at startup (s6 oneshot) before any service starts.
# Reads Docker secrets + env vars → generates Postfix/OpenDKIM/Mailpit config.
# =============================================================================
set -eu

LOG() { printf '[configure] %s\n' "$*" >&2; }

# ── Helper: read Docker secret ────────────────────────────────────────────────
read_secret() {
    local file="/run/secrets/${1}"
    if [ -f "$file" ]; then
        tr -d '\n' < "$file"
    else
        echo ""
    fi
}

# ── Helper: export env var into s6 container environment ─────────────────────
s6_export() {
    printf '%s' "$2" > "/var/run/s6/container_environment/${1}"
}

# ── Resolve values: Docker secret > environment variable > default ────────────
RELAY_PASS=$(read_secret smtp_relay_password)
[ -z "$RELAY_PASS" ] && RELAY_PASS="${POSTFIX_RELAY_PASS:-}"

DKIM_KEY_B64=$(read_secret dkim_private_key)
[ -z "$DKIM_KEY_B64" ] && DKIM_KEY_B64="${DKIM_PRIVATE_KEY_B64:-}"

TLS_CERT_B64=$(read_secret tls_cert)
TLS_KEY_B64=$(read_secret tls_key)

MAILPIT_AUTH=$(read_secret mailpit_auth)
[ -z "$MAILPIT_AUTH" ] && MAILPIT_AUTH="${MP_UI_AUTH:-}"

MAIL_MODE="${MAIL_MODE:-catch}"
DKIM_ENABLED="${DKIM_ENABLED:-false}"
POSTFIX_HOSTNAME="${POSTFIX_HOSTNAME:-mail.local}"
POSTFIX_MYORIGIN="${POSTFIX_MYORIGIN:-$POSTFIX_HOSTNAME}"
POSTFIX_RELAY_HOST="${POSTFIX_RELAY_HOST:-}"
POSTFIX_RELAY_USER="${POSTFIX_RELAY_USER:-}"
POSTFIX_MESSAGE_SIZE="${POSTFIX_MESSAGE_SIZE:-52428800}"
POSTFIX_MYNETWORKS="${POSTFIX_MYNETWORKS:-127.0.0.0/8}"
DKIM_DOMAIN="${DKIM_DOMAIN:-$POSTFIX_MYORIGIN}"
DKIM_SELECTOR="${DKIM_SELECTOR:-default}"
DKIM_KEY_SIZE="${DKIM_KEY_SIZE:-2048}"
TLS_ENABLED="${TLS_ENABLED:-false}"

LOG "MAIL_MODE=${MAIL_MODE}, DKIM_ENABLED=${DKIM_ENABLED}, TLS_ENABLED=${TLS_ENABLED}"

# =============================================================================
# Mailpit authentication
# =============================================================================
if [ -n "$MAILPIT_AUTH" ]; then
    printf '%s\n' "$MAILPIT_AUTH" > /run/mailpit-auth
    chmod 0600 /run/mailpit-auth          # chmod before chown (root still owns it)
    chown mailpit:mailpit /run/mailpit-auth
    LOG "Mailpit basic auth configured."
fi

# =============================================================================
# Mailpit SMTP relay → Postfix (only if sending is enabled)
# =============================================================================
if [ "$MAIL_MODE" != "catch" ]; then
    s6_export MP_SMTP_RELAY_HOST "127.0.0.1"
    s6_export MP_SMTP_RELAY_PORT "2525"
    s6_export MP_SMTP_RELAY_ALL  "true"
    LOG "Mailpit relay → Postfix:2525 enabled."
fi

# =============================================================================
# Postfix configuration (skip in catch mode)
# =============================================================================
if [ "$MAIL_MODE" = "catch" ]; then
    LOG "catch mode — Postfix not started."
else
    # ── TLS certificates ──────────────────────────────────────────────────────
    TLS_CERT_FILE=""
    TLS_KEY_FILE=""
    if [ "$TLS_ENABLED" = "true" ] && [ -n "$TLS_CERT_B64" ] && [ -n "$TLS_KEY_B64" ]; then
        TLS_CERT_FILE="/etc/postfix/tls/cert.pem"
        TLS_KEY_FILE="/etc/postfix/tls/key.pem"
        printf '%s' "$TLS_CERT_B64" | base64 -d > "$TLS_CERT_FILE"
        printf '%s' "$TLS_KEY_B64"  | base64 -d > "$TLS_KEY_FILE"
        chmod 0640 "$TLS_KEY_FILE"
        LOG "TLS certificates installed."
    fi

    # ── SASL relay credentials ────────────────────────────────────────────────
    RELAY_CONFIG=""
    if [ "$MAIL_MODE" = "relay" ] && [ -n "$POSTFIX_RELAY_HOST" ]; then
        RELAY_CONFIG="
# Relay host
relayhost = ${POSTFIX_RELAY_HOST}
smtp_sasl_auth_enable = yes
smtp_sasl_password_maps = hash:/etc/postfix/sasl/sasl_passwd
smtp_sasl_security_options = noanonymous
smtp_tls_security_level = encrypt
smtp_tls_mandatory_protocols = >=TLSv1.2
smtp_tls_protocols = >=TLSv1.2
smtp_tls_mandatory_ciphers = high
smtp_tls_ciphers = high
smtp_tls_wrappermode = no"
        printf '%s %s:%s\n' \
            "$POSTFIX_RELAY_HOST" \
            "$POSTFIX_RELAY_USER" \
            "$RELAY_PASS" \
            > /etc/postfix/sasl/sasl_passwd
        chmod 0600 /etc/postfix/sasl/sasl_passwd
        postmap /etc/postfix/sasl/sasl_passwd
        LOG "Relay configured → ${POSTFIX_RELAY_HOST}"
    elif [ "$MAIL_MODE" = "direct" ]; then
        RELAY_CONFIG="
# Direct delivery (DNS MX lookup)
relayhost =
smtp_tls_security_level = may
smtp_tls_protocols = >=TLSv1.2
smtp_tls_mandatory_protocols = >=TLSv1.2
smtp_tls_ciphers = high
smtp_tls_mandatory_ciphers = high
smtp_tls_note_starttls_offer = yes"
        LOG "Direct delivery mode (MX lookup)."
    fi

    # ── TLS config block ──────────────────────────────────────────────────────
    TLS_CONFIG=""
    if [ -n "$TLS_CERT_FILE" ]; then
        TLS_CONFIG="
# TLS
smtpd_tls_cert_file = ${TLS_CERT_FILE}
smtpd_tls_key_file  = ${TLS_KEY_FILE}
smtpd_tls_security_level = may
smtpd_tls_session_cache_database = btree:/var/lib/postfix/smtpd_scache
smtp_tls_session_cache_database  = btree:/var/lib/postfix/smtp_scache"
    fi

    # ── DKIM milter block ─────────────────────────────────────────────────────
    DKIM_CONFIG=""
    if [ "$DKIM_ENABLED" = "true" ]; then
        DKIM_CONFIG="
# OpenDKIM milter
milter_default_action = accept
milter_protocol = 6
smtpd_milters     = local:/run/opendkim/opendkim.sock
non_smtpd_milters = \$smtpd_milters"
    fi

    # ── Generate main.cf ──────────────────────────────────────────────────────
    cat > /etc/postfix/main.cf <<MAINCF
# Generated by configure-all.sh — do not edit manually.
# =============================================================================
# Identity
# =============================================================================
myhostname = ${POSTFIX_HOSTNAME}
myorigin   = ${POSTFIX_MYORIGIN}

# =============================================================================
# Delivery scope — local only (container receives from Mailpit on loopback)
# =============================================================================
mydestination      = localhost, \$myhostname
mynetworks         = ${POSTFIX_MYNETWORKS}
inet_interfaces    = loopback-only
inet_protocols     = ipv4
message_size_limit = ${POSTFIX_MESSAGE_SIZE}

# =============================================================================
# Queue
# =============================================================================
queue_directory   = /var/spool/postfix
command_directory = /usr/sbin
daemon_directory  = /usr/libexec/postfix
data_directory    = /var/lib/postfix
mail_owner        = postfix
compatibility_level = 3.6

# =============================================================================
# Logging → stdout (Docker-friendly)
# =============================================================================
maillog_file = /dev/stdout

# =============================================================================
# Security
# =============================================================================
disable_vrfy_command    = yes
smtpd_helo_required     = yes
strict_rfc821_envelopes = yes
smtpd_recipient_restrictions =
    permit_mynetworks,
    reject_unauth_destination
${RELAY_CONFIG}
${TLS_CONFIG}
${DKIM_CONFIG}
MAINCF

    # ── Aliases ───────────────────────────────────────────────────────────────
    touch /etc/postfix/aliases 2>/dev/null || true
    postalias /etc/postfix/aliases 2>/dev/null || true

    # ── Validate config ───────────────────────────────────────────────────────
    postfix check 2>&1 | while IFS= read -r line; do LOG "$line"; done || true
    LOG "Postfix configuration complete."
fi

# =============================================================================
# OpenDKIM configuration (skip if disabled or in catch mode)
# =============================================================================
if [ "$DKIM_ENABLED" = "true" ] && [ "$MAIL_MODE" != "catch" ]; then
    DKIM_KEY_DIR="/var/lib/opendkim/${DKIM_DOMAIN}"
    mkdir -p "$DKIM_KEY_DIR"

    if [ -n "$DKIM_KEY_B64" ]; then
        printf '%s' "$DKIM_KEY_B64" | base64 -d > "${DKIM_KEY_DIR}/private.key"
        LOG "DKIM private key installed from secret."
    else
        LOG "No DKIM key provided — auto-generating ${DKIM_KEY_SIZE}-bit key..."
        opendkim-genkey -b "$DKIM_KEY_SIZE" -D "$DKIM_KEY_DIR" -d "$DKIM_DOMAIN" -s "$DKIM_SELECTOR"
        mv "${DKIM_KEY_DIR}/${DKIM_SELECTOR}.private" "${DKIM_KEY_DIR}/private.key" 2>/dev/null || true
        LOG "Auto-generated DKIM key. Add this DNS TXT record:"
        cat "${DKIM_KEY_DIR}/${DKIM_SELECTOR}.txt" 2>/dev/null >&2 || true
    fi

    # chmod BEFORE chown — root can chmod its own files but not after chown without FOWNER
    chmod 0600 "${DKIM_KEY_DIR}/private.key" 2>/dev/null || true
    chown -R opendkim:opendkim "$DKIM_KEY_DIR" /run/opendkim 2>/dev/null || true

    cat > /etc/opendkim.conf <<DKIMCONF
# Generated by configure-all.sh
Syslog          yes
SyslogSuccess   yes
LogWhy          yes
Canonicalization relaxed/simple
Domain          ${DKIM_DOMAIN}
Selector        ${DKIM_SELECTOR}
KeyFile         ${DKIM_KEY_DIR}/private.key
Socket          local:/run/opendkim/opendkim.sock
PidFile         /run/opendkim/opendkim.pid
UMask           0
OversignHeaders From
Mode            sv
SubDomains      no
DKIMCONF

    LOG "OpenDKIM configured for domain=${DKIM_DOMAIN} selector=${DKIM_SELECTOR}."
fi

LOG "Startup configuration complete."
