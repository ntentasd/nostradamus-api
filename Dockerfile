FROM golang:1.24.3-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o api ./cmd/api/main.go

FROM alpine:latest

WORKDIR /nostradamus

COPY --from=build /app/api .

ENTRYPOINT [ "/nostradamus/api" ]