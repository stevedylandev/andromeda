# Build from repo root: docker build -t bookmarks-go -f apps/bookmarks-go/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/bookmarks-go/go.mod apps/bookmarks-go/go.sum ./apps/bookmarks-go/
WORKDIR /app/apps/bookmarks-go
RUN go mod download
COPY apps/bookmarks-go/ ./
RUN CGO_ENABLED=0 go build -o /bookmarks-go .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /bookmarks-go /usr/local/bin/bookmarks-go
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["bookmarks-go"]
