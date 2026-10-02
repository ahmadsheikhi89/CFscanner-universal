FROM golang:1.25.7-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build \
    -buildvcs=false \
    -trimpath \
    -ldflags="-s -w -buildid=" \
    -o /out/cfscanner \
    .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       ca-certificates \
       tzdata \
       python3 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/cfscanner /usr/local/bin/cfscanner
COPY assets/cf.local.iplist /app/cf.local.iplist
COPY docker/entrypoint.sh /usr/local/bin/cfscanner-menu

RUN chmod +x /usr/local/bin/cfscanner-menu

RUN mkdir -p /data

VOLUME ["/data"]

LABEL org.opencontainers.image.title="CFScanner Universal"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.description="Cloudflare IP scanner for VLESS, VMess and Trojan"
LABEL org.opencontainers.image.authors="Ahmad Sheikhi <Senior Platform / DevOps Engineer>"
LABEL org.opencontainers.image.source="Derived from ArchNets/CDN-IP-Scanner"

ENTRYPOINT ["cfscanner-menu"]
