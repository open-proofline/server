FROM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

WORKDIR /src

RUN apk add --no-cache build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/safety-recorder ./cmd/api

FROM alpine:3.22@sha256:310c62b5e7ca5b08167e4384c68db0fd2905dd9c7493756d356e893909057601

RUN apk add --no-cache ca-certificates tzdata \
	&& addgroup -S safety \
	&& adduser -S -G safety -h /nonexistent -s /sbin/nologin safety \
	&& mkdir -p /data \
	&& chown -R safety:safety /data

ENV SAFE_PRIVATE_BIND_ADDRS=0.0.0.0:8080 \
	SAFE_PUBLIC_BIND_ADDRS=0.0.0.0:8081 \
	SAFE_DATA_DIR=/data \
	SAFE_DB_PATH=/data/safety.db

COPY --from=builder /out/safety-recorder /usr/local/bin/safety-recorder

USER safety
WORKDIR /data
VOLUME ["/data"]
EXPOSE 8080 8081

ENTRYPOINT ["safety-recorder"]
