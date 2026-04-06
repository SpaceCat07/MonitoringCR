# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/monitoringcr ./main.go

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/monitoringcr /app/monitoringcr

EXPOSE 8080

ENTRYPOINT ["/app/monitoringcr"]
