# Build from repo root: docker build -t easel-go -f apps/easel-go/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/easel-go/go.mod apps/easel-go/go.sum ./apps/easel-go/
WORKDIR /app/apps/easel-go
RUN go mod download
COPY apps/easel-go/ ./
RUN CGO_ENABLED=0 go build -o /easel-go .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
COPY --from=builder /easel-go /usr/local/bin/easel-go
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=4242
EXPOSE 4242
CMD ["easel-go"]
