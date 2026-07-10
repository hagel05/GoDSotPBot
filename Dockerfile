# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.22-alpine AS build
WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

# Build a static binary (no libc dependency) so it runs on a distroless base.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-s -w" \
    -o /out/bot ./cmd/bot

# ---- Runtime stage ----
# distroless/static has no shell and runs as nonroot; ships no config or secrets.
FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/bot /bot

# League data is provided at runtime via a mounted volume, not baked in.
ENV PORT=8080 \
    LEAGUE_CONFIG_PATH=/config/league.yaml

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/bot"]
