# Build from repo root: docker build -t bookmarks -f apps/bookmarks/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY apps/bookmarks/go.mod apps/bookmarks/go.sum ./apps/bookmarks/
WORKDIR /app/apps/bookmarks
RUN go mod download
COPY apps/bookmarks/ ./
RUN CGO_ENABLED=0 go build -o /bookmarks .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /bookmarks /usr/local/bin/bookmarks
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["bookmarks"]
