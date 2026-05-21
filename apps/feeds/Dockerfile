# Build from repo root: docker build -t feeds -f apps/feeds/Dockerfile .
FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY apps/feeds/go.mod apps/feeds/go.sum ./apps/feeds/
WORKDIR /app/apps/feeds
RUN go mod download
COPY apps/feeds/ ./
RUN CGO_ENABLED=0 go build -o /feeds .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /feeds /usr/local/bin/feeds
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
ENV GODEBUG=x509usefallbackroots=1
EXPOSE 3000
CMD ["feeds"]
