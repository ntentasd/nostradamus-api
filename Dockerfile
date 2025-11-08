FROM migrate/migrate:latest AS migrate-cli

FROM golang:1.24.3-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o api ./cmd/api/main.go

FROM alpine:latest

# Copy migrate cli
COPY --from=migrate-cli /usr/local/bin/migrate /usr/local/bin

WORKDIR /opt/migrations

# Copy migrations
COPY migrations/ .

WORKDIR /nostradamus

# Copy binary
COPY --from=build /app/api .

ENTRYPOINT [ "/nostradamus/api" ]