# Build from repo root: docker build -t library-go -f apps/library-go/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/library-go/go.mod apps/library-go/go.sum ./apps/library-go/
WORKDIR /app/apps/library-go
RUN go mod download
COPY apps/library-go/ ./
RUN CGO_ENABLED=0 go build -o /library-go .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /library-go /usr/local/bin/library-go
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["library-go"]
