# Build from repo root: docker build -t cellar -f apps/cellar/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/cellar/go.mod apps/cellar/go.sum ./apps/cellar/
WORKDIR /app/apps/cellar
RUN go mod download
COPY apps/cellar/ ./
RUN CGO_ENABLED=0 go build -o /cellar .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /cellar /usr/local/bin/cellar
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["cellar"]
