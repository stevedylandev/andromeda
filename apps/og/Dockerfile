# Build from repo root: docker build -t og -f apps/og/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY apps/og/go.mod apps/og/go.sum ./apps/og/
WORKDIR /app/apps/og
RUN go mod download
COPY apps/og/ ./
RUN CGO_ENABLED=0 go build -o /og .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /og /usr/local/bin/og
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["og"]
