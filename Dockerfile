FROM golang:1.27-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS builder

WORKDIR /src

RUN apk add --no-cache build-base

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/proofline-server ./cmd/api

FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

RUN apk add --no-cache ca-certificates tzdata \
	&& addgroup -S proofline \
	&& adduser -S -G proofline -h /nonexistent -s /sbin/nologin proofline \
	&& mkdir -p /var/lib/proofline /etc/proofline \
	&& chown -R proofline:proofline /var/lib/proofline

COPY --from=builder /out/proofline-server /usr/local/bin/proofline-server
COPY docker-default-config.toml /etc/proofline/proofline.toml

USER proofline
WORKDIR /var/lib/proofline
VOLUME ["/var/lib/proofline"]
EXPOSE 8080 8081

ENTRYPOINT ["proofline-server", "--config", "/etc/proofline/proofline.toml"]
