# Build from repo root: docker build -t jotts -f apps/jotts/Dockerfile .
FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY apps/jotts/go.mod apps/jotts/go.sum ./apps/jotts/
WORKDIR /app/apps/jotts
RUN go mod download
COPY apps/jotts/ ./
RUN CGO_ENABLED=0 go build -o /jotts .

FROM debian:bookworm-slim
COPY --from=builder /jotts /usr/local/bin/jotts
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["jotts", "server"]
