# Build from repo root: docker build -t posts -f apps/posts/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY apps/posts/go.mod apps/posts/go.sum ./apps/posts/
WORKDIR /app/apps/posts
RUN go mod download
COPY apps/posts/ ./
RUN CGO_ENABLED=0 go build -o /posts .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /posts /usr/local/bin/posts
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
ENV UPLOADS_DIR=/data/uploads
EXPOSE 3000
CMD ["posts"]
