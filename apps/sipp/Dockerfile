# Build from repo root: docker build -t sipp -f apps/sipp/Dockerfile .
FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/sipp/go.mod apps/sipp/go.sum ./apps/sipp/
WORKDIR /app/apps/sipp
RUN go mod download
COPY apps/sipp/ ./
RUN CGO_ENABLED=0 go build -o /sipp .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /sipp /usr/local/bin/sipp
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["sipp", "server"]
