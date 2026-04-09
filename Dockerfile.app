# ═══════════════════════════════════════════════════════════════════════════
# MD – Multi-stage Dockerfile
# ═══════════════════════════════════════════════════════════════════════════
#
# This Dockerfile produces a single container (~150 MB) that serves both
# the Go REST API and the SvelteKit SPA, with Pandoc + WeasyPrint for
# multi-format document export (PDF, DOCX, HTML, etc.).
#
# Build stages:
#   1. web-build  → Compile the SvelteKit frontend (Node 24)
#   2. go-build   → Compile the Go binary with embedded metadata (Go 1.25)
#   3. runtime    → Minimal Alpine image with Pandoc + WeasyPrint + fonts
#
# Build args (set via docker-compose or CI):
#   VERSION    → displayed on /health endpoint (e.g. "1.2.3")
#   GIT_SHA    → git commit hash for traceability
#   BUILD_DATE → ISO-8601 build timestamp
#
# Usage:
#   docker build -t md-app --build-arg VERSION=1.0.0 -f Dockerfile.app .
#
# ═══════════════════════════════════════════════════════════════════════════


# ─────────────────────────────────────────────────────────────
# Stage 1: Build SvelteKit frontend
# ─────────────────────────────────────────────────────────────
# Produces a static SPA in /src/web/dist/ (adapter-static).
# Only package.json + lock file are copied first for layer caching.
FROM node:24-alpine AS web-build
WORKDIR /src

# Install dependencies (cached unless package*.json changes)
COPY web/package*.json ./web/
RUN cd web && npm ci --prefer-offline

# Copy source and build the SPA
COPY web/ ./web/
RUN cd web && npm run build


# ─────────────────────────────────────────────────────────────
# Stage 1b: Build mermaid SSR renderer
# ─────────────────────────────────────────────────────────────
# Installs mermaid + happy-dom for server-side SVG rendering.
# Used by the Go export pipeline to render mermaid diagrams in PDFs.
FROM node:24-alpine AS mermaid-build
WORKDIR /mmdc
COPY pandoc/mermaid-ssr/package.json ./
# Skip Puppeteer's bundled Chromium download; we'll use Alpine's chromium
ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD=true
RUN npm install --omit=dev
COPY pandoc/mermaid-ssr/render.mjs pandoc/mermaid-ssr/puppeteer.json pandoc/mermaid-ssr/mermaid.config.json ./


# ─────────────────────────────────────────────────────────────
# Stage 2: Build Go binary
# ─────────────────────────────────────────────────────────────
# Produces a statically-linked binary at /app/md (~15 MB).
# CGO is disabled for a fully static build (no libc dependency).
FROM golang:1.26-alpine AS go-build
WORKDIR /src

# Ensure the correct Go toolchain is used
ENV GOTOOLCHAIN=auto

# Static build: no CGO
ENV CGO_ENABLED=0

# Git is needed for go mod download (private repos or git-based deps)
RUN apk add --no-cache git

# Download Go dependencies (cached unless go.mod/go.sum changes)
COPY go.mod go.sum ./
RUN go mod download

# Copy full source tree
COPY . .

# Run unit tests during build
RUN go test -short ./...

# Build args injected as linker flags into the binary
ARG VERSION=dev
ARG GIT_SHA=unknown
ARG BUILD_DATE=unknown

# Build the binary with:
#   -s -w     → strip debug info (smaller binary)
#   -X main.* → inject version metadata at compile time
#   -trimpath → reproducible builds (remove local paths from binary)
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.GitSHA=${GIT_SHA} -X main.BuildDate=${BUILD_DATE}" \
    -trimpath \
    -o /app/md \
    ./cmd/server


# ─────────────────────────────────────────────────────────────
# Stage 3: Runtime image (Alpine + Pandoc + WeasyPrint)
# ─────────────────────────────────────────────────────────────
# Minimal production image. Only the compiled binary, the SPA,
# Pandoc templates, and system tools are included.
FROM alpine:3.21 AS runtime
# Note: upgrade to alpine:3.23 when available and tested

# OCI image metadata (adjust source URL if you forked the project)
LABEL org.opencontainers.image.title="MD"
LABEL org.opencontainers.image.description="Open-source markdown editor & file manager"
LABEL org.opencontainers.image.source="https://github.com/cybergraphe-fr/md-app"
LABEL org.opencontainers.image.licenses="GPL-3.0"

# System dependencies:
#   pandoc          → multi-format export (DOCX, HTML, LaTeX, etc.)
#   py3-weasyprint  → HTML→PDF conversion (used by export pipeline)
#   ca-certificates → TLS for outbound HTTPS (OIDC, webhooks)
#   tzdata          → timezone support (TZ env var)
#   font-dejavu     → fallback serif/sans/mono fonts for PDF rendering
#   font-liberation → metric-compatible alternatives to Arial/Times/Courier
#   ttf-liberation  → TrueType version of Liberation fonts
#   font-noto-emoji → Noto Emoji glyphs for PDF emoji rendering (WeasyPrint)
#   nodejs          → Required for mermaid SSR (PDF diagram rendering)
#   chromium        → Headless browser for mermaid-cli SVG rendering
RUN apk add --no-cache \
    pandoc \
    py3-weasyprint \
    ca-certificates \
    tzdata \
    font-dejavu \
    font-liberation \
    ttf-liberation \
    font-noto-emoji \
    nodejs \
    chromium \
    && rm -rf /var/cache/apk/*

# Tell Puppeteer to use system-installed Chromium
ENV PUPPETEER_EXECUTABLE_PATH=/usr/bin/chromium-browser

# Fontconfig: restrict Noto Color Emoji charset to emoji-only codepoints.
# Without this, fontconfig uses Noto Color Emoji as a general fallback and
# its oversized keycap-digit glyphs break spacing for regular numbers.
# The font stays fully accessible by name for .emoji CSS class usage.
RUN mkdir -p /etc/fonts/conf.d && printf '\
<?xml version="1.0"?>\n\
<!DOCTYPE fontconfig SYSTEM "urn:fontconfig:fonts.dtd">\n\
<fontconfig>\n\
  <match target="scan">\n\
    <test name="family" compare="contains"><string>Emoji</string></test>\n\
    <edit name="charset" mode="assign">\n\
      <charset>\n\
        <range><int>0x200D</int><int>0x200D</int></range>\n\
        <range><int>0x2049</int><int>0x2B55</int></range>\n\
        <range><int>0x2600</int><int>0x27BF</int></range>\n\
        <range><int>0x2934</int><int>0x2935</int></range>\n\
        <range><int>0x3030</int><int>0x3030</int></range>\n\
        <range><int>0x303D</int><int>0x303D</int></range>\n\
        <range><int>0xFE00</int><int>0xFE0F</int></range>\n\
        <range><int>0x1F000</int><int>0x1FBFF</int></range>\n\
        <range><int>0xE0020</int><int>0xE007F</int></range>\n\
      </charset>\n\
    </edit>\n\
  </match>\n\
</fontconfig>\n' > /etc/fonts/conf.d/99-emoji-charset.conf \
    && fc-cache -f

# Security: run as non-root user "md" with fixed UID/GID 1000.
# This matches the default NAS user (pianographe) for bind-mount compatibility.
RUN addgroup -g 1000 md && adduser -u 1000 -G md -D md

# Create app and data directories with correct ownership
# /data/files  → user markdown files
# /data/.meta  → file metadata (JSON sidecar files)
RUN mkdir -p /app /data/files /data/.meta && \
    chown -R md:md /app /data

WORKDIR /app

# Copy artifacts from build stages
COPY --from=go-build /app/md ./md
COPY --from=web-build /src/web/dist ./web
# Pandoc templates + print.css for PDF/DOCX export
COPY pandoc/ ./pandoc/
# Mermaid SSR renderer (Node.js) for PDF diagram rendering
COPY --from=mermaid-build /mmdc ./mmdc

# Switch to non-root user for all runtime operations
USER md

# Default environment variables (can be overridden in docker-compose)
ENV MD_HTTP_ADDR=:8080
ENV MD_STORAGE_PATH=/data
ENV MD_PANDOC_BINARY=pandoc
ENV TZ=UTC

# The Go server listens on this port
EXPOSE 8080

# Health check: the /health endpoint returns {"status":"ok","version":"..."}
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

# Start the Go server (no shell wrapper needed)
ENTRYPOINT ["./md"]
