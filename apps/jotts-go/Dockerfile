# Build from repo root: docker build -t jotts-go -f apps/jotts-go/Dockerfile .
FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/jotts-go/go.mod apps/jotts-go/go.sum ./apps/jotts-go/
WORKDIR /app/apps/jotts-go
RUN go mod download
COPY apps/jotts-go/ ./
RUN CGO_ENABLED=0 go build -o /jotts-go .

FROM debian:bookworm-slim
COPY --from=builder /jotts-go /usr/local/bin/jotts-go
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["jotts-go"]
