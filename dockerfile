# syntax=docker/dockerfile:1
# Build from repo root: docker build -f dockerfile -t pc-armin/local-app-manager-api:latest .

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl
WORKDIR /app
COPY --from=build /out/server /app/server
ENV HTTP_ADDR=:8195
EXPOSE 8195
HEALTHCHECK --interval=5s --timeout=5s --retries=10 \
  CMD curl -fsS http://127.0.0.1:8195/health || exit 1
CMD ["/app/server"]
