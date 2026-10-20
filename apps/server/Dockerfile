# croffle-server image.
#
# Distribution is this image only: the repository's GitHub Releases belong to
# the desktop app, and electron-updater resolves "latest release" across the
# whole repository - a server release there would stop desktop updates.

# --- dashboard ---
FROM node:24-alpine AS web
WORKDIR /web

RUN corepack enable
# Its own pnpm root with its own lockfile.
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

COPY web/ ./
RUN pnpm build

# --- server ---
FROM golang:1.26-alpine AS build
WORKDIR /src

# Dependencies first, so a code change does not re-download them.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# The dashboard has to be in place before the build: //go:embed reads it at
# compile time.
COPY --from=web /web/embedded ./web/embedded

ARG VERSION=dev
ARG COMMIT=unknown
# CGO off: modernc.org/sqlite is pure Go, which is what makes a static binary
# and a trivial cross-compile possible.
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/croffle-server ./cmd/croffle-server

# --- runtime ---
FROM alpine:3.21

# wget is in busybox and is what HEALTHCHECK uses; ca-certificates lets an
# operator's webhook target be an https URL.
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S croffle && adduser -S -G croffle croffle \
    && mkdir -p /var/lib/croffle && chown croffle:croffle /var/lib/croffle

COPY --from=build /out/croffle-server /usr/local/bin/croffle-server

USER croffle
WORKDIR /var/lib/croffle
VOLUME ["/var/lib/croffle"]
EXPOSE 8787

ENV CROFFLE_LISTEN=:8787 \
    CROFFLE_DB_PATH=/var/lib/croffle/server.db

# /health is public and cheap; it reports 503 when the database is unreachable.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O- http://127.0.0.1:8787/health >/dev/null || exit 1

ENTRYPOINT ["croffle-server"]
