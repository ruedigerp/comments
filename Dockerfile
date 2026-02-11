FROM --platform=$BUILDPLATFORM golang:1.26 AS builder

ARG VERSION
ARG STAGE
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Go Module Cache für bessere Layer-Caching
COPY go.mod go.sum ./
# RUN go mod download
RUN go mod tidy

# Source Code kopieren
COPY . .

# Multi-Arch Build mit Build-Args
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags "-X main.version=${VERSION} -X main.stage=${STAGE} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -w -s" \
    -o server main.go

# --- Runtime Stage ---
FROM --platform=$TARGETPLATFORM alpine:latest

# Timezone und CA Certificates für HTTPS
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

ENV TZ=Europe/Berlin

# User für Security
RUN addgroup -g 1001 appgroup && \
    adduser -u 1001 -G appgroup -s /bin/sh -D appuser

# Binary kopieren
COPY --from=builder /app/server /app/server
COPY --from=builder /app/static /app/static
COPY --from=builder /app/templates /app/templates

# Ownership an appuser geben
RUN chown -R appuser:appgroup /app
USER appuser

EXPOSE 8080

# Health Check
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#     CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/comments || exit 1

CMD ["./server"]
