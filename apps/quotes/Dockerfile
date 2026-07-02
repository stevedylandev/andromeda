# Build from repo root: docker build -t quotes -f apps/quotes/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY apps/quotes/go.mod apps/quotes/go.sum ./apps/quotes/
WORKDIR /app/apps/quotes
RUN go mod download
COPY apps/quotes/ ./
RUN CGO_ENABLED=0 go build -o /quotes .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /quotes /usr/local/bin/quotes
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["quotes"]
