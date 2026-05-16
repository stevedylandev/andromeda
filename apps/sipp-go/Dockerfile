# Build from repo root: docker build -t sipp-go -f apps/sipp-go/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/sipp-go/go.mod apps/sipp-go/go.sum ./apps/sipp-go/
WORKDIR /app/apps/sipp-go
RUN go mod download
COPY apps/sipp-go/ ./
RUN CGO_ENABLED=0 go build -o /sipp-server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /sipp-server /usr/local/bin/sipp-server
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["sipp-server"]
