# Build from repo root: docker build -t blobs -f apps/blobs/Dockerfile .
FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY apps/blobs/go.mod apps/blobs/go.sum ./apps/blobs/
WORKDIR /app/apps/blobs
RUN go mod download
COPY apps/blobs/ ./
RUN CGO_ENABLED=0 go build -o /blobs .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /blobs /usr/local/bin/blobs
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
ENV BLOBS_DB_PATH=/data/blobs.sqlite
EXPOSE 3000
CMD ["blobs", "server"]
