# Build from repo root: docker build -t library -f apps/library/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/library/go.mod apps/library/go.sum ./apps/library/
WORKDIR /app/apps/library
RUN go mod download
COPY apps/library/ ./
RUN CGO_ENABLED=0 go build -o /library .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /library /usr/local/bin/library
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["library"]
