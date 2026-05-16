# Build from repo root: docker build -t og-go -f apps/og-go/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/og-go/go.mod apps/og-go/go.sum ./apps/og-go/
WORKDIR /app/apps/og-go
RUN go mod download
COPY apps/og-go/ ./
RUN CGO_ENABLED=0 go build -o /og-go .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /og-go /usr/local/bin/og-go
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["og-go"]
