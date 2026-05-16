# Build from repo root: docker build -t feeds-go -f apps/feeds-go/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/feeds-go/go.mod apps/feeds-go/go.sum ./apps/feeds-go/
WORKDIR /app/apps/feeds-go
RUN go mod download
COPY apps/feeds-go/ ./
RUN CGO_ENABLED=0 go build -o /feeds-go .

FROM debian:bookworm-slim
COPY --from=builder /feeds-go /usr/local/bin/feeds-go
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["feeds-go"]
