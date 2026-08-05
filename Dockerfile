# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder
WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' -o /out/frantic-postr ./

FROM alpine:3.21
WORKDIR /data

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/frantic-postr /app/frantic-postr
COPY --from=builder /src/res /app/res
COPY version.no /app/version.no
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh

# Seed data copied into mounted folders only when destination is empty.
COPY config /seed/config
COPY templates /seed/templates
COPY fonts /seed/fonts
RUN mkdir -p /seed/output /seed/backups /seed/logs /app && chmod +x /usr/local/bin/entrypoint.sh /app/frantic-postr

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["/app/frantic-postr", "-config", "/data/config/config.toml", "-web", "-port", "8080"]
