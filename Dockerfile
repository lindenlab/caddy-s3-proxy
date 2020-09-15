FROM caddy:2-builder AS builder

RUN caddy-builder \
    github.com/lindenlab/caddy-s3-proxy@v0.1.1-pre.2

FROM caddy:latest

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
