# Build from repo root: docker build -t easel -f apps/easel/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY apps/easel/go.mod apps/easel/go.sum ./apps/easel/
WORKDIR /app/apps/easel
RUN go mod download
COPY apps/easel/ ./
RUN CGO_ENABLED=0 go build -o /easel .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
COPY --from=builder /easel /usr/local/bin/easel
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=4242
EXPOSE 4242
CMD ["easel"]
