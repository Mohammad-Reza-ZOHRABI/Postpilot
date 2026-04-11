# =============================================================================
# Postpilot — Hardened single-container image
# Pilot UI (Svelte) + Mailpit (SMTP) + Postfix (MTA) + OpenDKIM (DKIM)
# Managed by s6-overlay · Base: Alpine 3.23
# =============================================================================

# =============================================================================
# Stage 1: Download s6-overlay (multi-arch)
# =============================================================================
FROM alpine:3.23 AS s6-download

ARG S6_VERSION=3.2.0.2
ARG TARGETARCH

RUN apk add --no-cache wget xz \
 && case "${TARGETARCH}" in \
      arm64) ARCH=aarch64 ;; \
      *)     ARCH=x86_64  ;; \
    esac \
 && wget -q "https://github.com/just-containers/s6-overlay/releases/download/v${S6_VERSION}/s6-overlay-noarch.tar.xz" \
 && wget -q "https://github.com/just-containers/s6-overlay/releases/download/v${S6_VERSION}/s6-overlay-${ARCH}.tar.xz" \
 && mkdir -p /s6-rootfs \
 && tar -C /s6-rootfs -Jxpf s6-overlay-noarch.tar.xz \
 && tar -C /s6-rootfs -Jxpf "s6-overlay-${ARCH}.tar.xz"

# =============================================================================
# Stage 2: Download Mailpit static binary (multi-arch)
# =============================================================================
FROM alpine:3.23 AS mailpit-download

ARG MAILPIT_VERSION=1.21.5
ARG TARGETARCH

RUN apk add --no-cache wget \
 && case "${TARGETARCH}" in \
      arm64) ARCH=arm64 ;; \
      *)     ARCH=amd64 ;; \
    esac \
 && wget -q -O /tmp/mailpit.tar.gz \
      "https://github.com/axllent/mailpit/releases/download/v${MAILPIT_VERSION}/mailpit-linux-${ARCH}.tar.gz" \
 && tar -C /tmp -xzf /tmp/mailpit.tar.gz mailpit \
 && chmod 0755 /tmp/mailpit

# =============================================================================
# Stage 3: Build Svelte UI
# =============================================================================
FROM node:25-alpine AS ui-build

WORKDIR /ui
COPY pilot/ui/package*.json ./
RUN npm ci --ignore-scripts
COPY pilot/ui/ ./
RUN npm run build

# =============================================================================
# Stage 4: Build Pilot backend (Go static binary)
# =============================================================================
FROM golang:1.26-alpine AS pilot-build

WORKDIR /build
COPY pilot/ ./
# Copy built Svelte UI into web/dist/ for Go embed
COPY --from=ui-build /ui/dist ./web/dist/
# Resolve dependencies, scan for known Go CVEs, then build a static binary.
# govulncheck will fail the build if any direct or transitive Go dependency
# has a known vulnerability reachable from the code. To bypass temporarily
# during a CVE disclosure window, replace `govulncheck ./...` with
# `govulncheck ./... || true` (not recommended for long).
RUN go mod tidy \
 && go install golang.org/x/vuln/cmd/govulncheck@latest \
 && govulncheck ./... \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o /pilot .

# =============================================================================
# Stage 5: Production image
# =============================================================================
FROM alpine:3.23

ARG S6_VERSION=3.2.0.2
ARG MAILPIT_VERSION=1.21.5
ARG IMAGE_VERSION=1.0.0
ARG IMAGE_VENDOR=Postpilot
ARG IMAGE_SOURCE=https://github.com/Mohammad-Reza-ZOHRABI/Postpilot

LABEL org.opencontainers.image.title="Postpilot" \
      org.opencontainers.image.description="Hardened Mailpit + Postfix + OpenDKIM with admin UI" \
      org.opencontainers.image.vendor="${IMAGE_VENDOR}" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.source="${IMAGE_SOURCE}" \
      org.opencontainers.image.version="${IMAGE_VERSION}" \
      postpilot.s6_version="${S6_VERSION}" \
      postpilot.mailpit_version="${MAILPIT_VERSION}"

RUN apk add --no-cache \
      postfix postfix-pcre cyrus-sasl \
      opendkim opendkim-utils \
      ca-certificates curl \
 && rm -rf /var/cache/apk/* /usr/share/doc /usr/share/man /usr/share/info \
 && find /usr/share/locale -mindepth 1 -maxdepth 1 ! -name 'en*' -exec rm -rf {} + 2>/dev/null || true

COPY --from=s6-download /s6-rootfs/ /
COPY --from=mailpit-download /tmp/mailpit /usr/local/bin/mailpit
COPY --from=pilot-build /pilot /usr/local/bin/pilot
COPY rootfs/ /

RUN addgroup -g 10001 mailpit \
 && adduser -u 10001 -G mailpit -s /usr/sbin/nologin -D -H mailpit \
 && addgroup -g 10002 pilot \
 && adduser -u 10002 -G pilot -s /usr/sbin/nologin -D -H pilot

RUN mkdir -p /data/mailpit /data/postpilot /run/opendkim /var/lib/opendkim \
      /etc/postfix/sasl /etc/postfix/tls \
 && chown -R mailpit:mailpit /data/mailpit \
 && chown -R pilot:pilot /data/postpilot \
 && chown root:opendkim /var/lib/opendkim && chmod 0775 /var/lib/opendkim \
 && chown opendkim:opendkim /run/opendkim \
 && chmod 0750 /etc/postfix/sasl /etc/postfix/tls \
 && chown root:pilot /etc/postfix/main.cf && chmod 0664 /etc/postfix/main.cf \
 && find /etc/s6-overlay -type f \( -name "run" -o -name "finish" -o -name "up" -o -name "down" \) \
      -exec chmod 0755 {} + \
 && find /etc/s6-overlay/scripts -type f -exec chmod 0755 {} +

ENV MAIL_MODE=catch \
    POSTFIX_HOSTNAME=mail.local \
    POSTFIX_MYORIGIN=mail.local \
    DKIM_ENABLED=false \
    DKIM_SELECTOR=default \
    TLS_ENABLED=false \
    MP_SMTP_PORT=1025 \
    MP_UI_PORT=8025 \
    MP_MAX_MESSAGES=500 \
    PILOT_PORT=3000

VOLUME ["/data/mailpit", "/data/postpilot"]
EXPOSE 1025 3000 8025

HEALTHCHECK --interval=15s --timeout=5s --retries=5 --start-period=25s \
  CMD sh -c \
    'curl -sf http://127.0.0.1:${PILOT_PORT:-3000}/api/v1/health > /dev/null \
     && /usr/local/bin/mailpit readyz \
     && { [ "${MAIL_MODE:-catch}" = "catch" ] || postfix status > /dev/null 2>&1; }'

ENTRYPOINT ["/init"]
