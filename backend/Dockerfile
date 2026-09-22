# The go directive in go.mod pins 1.25.11, so the builder must be at least that.
FROM golang:1.25-alpine AS build

WORKDIR /src

# Dependencies are their own layer: source edits are frequent, go.mod changes are
# not, so a rebuild after a code change does not re-download the module graph.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.22

# The API opens TLS connections to nothing today, but a Postgres URL with
# sslmode=require is one environment variable away and fails obscurely without
# root certificates.
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=build /out/server /app/server
# Migrations are read from disk at startup, so they ship in the image beside the
# binary rather than being mounted in.
COPY migrations /app/migrations

ENV MIGRATIONS_PATH=/app/migrations

# Nothing in here needs to write to the filesystem or bind a privileged port.
USER nobody

EXPOSE 8080
ENTRYPOINT ["/app/server"]
