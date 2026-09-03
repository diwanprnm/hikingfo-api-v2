# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.22-alpine AS build
WORKDIR /src

# Cache module downloads first.
COPY backend/go.mod backend/go.sum* ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build /out/api /app/api
COPY backend/migrations/ /app/migrations/

EXPOSE 8080
ENV MIGRATIONS_DIR=/app/migrations
ENTRYPOINT ["/app/api"]
