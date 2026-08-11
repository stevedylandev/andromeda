# Build from repo root: docker build -t habbits -f apps/habbits/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY apps/habbits/go.mod apps/habbits/go.sum ./apps/habbits/
WORKDIR /app/apps/habbits
RUN go mod download
COPY apps/habbits/ ./
RUN CGO_ENABLED=0 go build -o /habbits .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /habbits /usr/local/bin/habbits
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["habbits"]
