FROM golang:1.25.3 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
ARG COMMIT_HASH=unknown
ARG CGO_ENABLED=0
ARG LDFLAGS="-X 'github.com/younsl/ghes-schedule-scanner/pkg/version.version=${VERSION}' \
             -X 'github.com/younsl/ghes-schedule-scanner/pkg/version.buildDate=${BUILD_DATE}' \
             -X 'github.com/younsl/ghes-schedule-scanner/pkg/version.gitCommit=${COMMIT_HASH}'"

RUN echo "Building with LDFLAGS: ${LDFLAGS}" && \
    go build -ldflags="${LDFLAGS}" -o ghes-schedule-scanner ./cmd/ghes-schedule-scanner

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /app/ghes-schedule-scanner .

ENTRYPOINT ["./ghes-schedule-scanner"]