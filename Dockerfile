FROM golang:1.21 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG CGO_ENABLED=0
RUN go build -o ghes-schedule-scanner ./cmd/ghes-schedule-scanner

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/ghes-schedule-scanner .

ENTRYPOINT ["./ghes-schedule-scanner"]