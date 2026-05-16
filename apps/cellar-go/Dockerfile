# Build from repo root: docker build -t cellar-go -f apps/cellar-go/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/cellar-go/go.mod apps/cellar-go/go.sum ./apps/cellar-go/
WORKDIR /app/apps/cellar-go
RUN go mod download
COPY apps/cellar-go/ ./
RUN CGO_ENABLED=0 go build -o /cellar-go .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /cellar-go /usr/local/bin/cellar-go
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["cellar-go"]
