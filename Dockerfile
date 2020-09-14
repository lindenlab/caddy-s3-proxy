FROM caddy:2-builder AS builder

RUN caddy-builder \
    github.com/lindenlab/caddy-s3-proxy

FROM caddy:latest

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
