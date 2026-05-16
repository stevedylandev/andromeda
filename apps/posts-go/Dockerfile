# Build from repo root: docker build -t posts-go -f apps/posts-go/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/posts-go/go.mod apps/posts-go/go.sum ./apps/posts-go/
WORKDIR /app/apps/posts-go
RUN go mod download
COPY apps/posts-go/ ./
RUN CGO_ENABLED=0 go build -o /posts-go .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /posts-go /usr/local/bin/posts-go
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
ENV UPLOADS_DIR=/data/uploads
EXPOSE 3000
CMD ["posts-go"]
