# Build from repo root: docker build -t shrink -f apps/shrink/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY apps/shrink/go.mod apps/shrink/go.sum ./apps/shrink/
WORKDIR /app/apps/shrink
RUN go mod download
COPY apps/shrink/ ./
RUN CGO_ENABLED=0 go build -o /shrink .

FROM debian:bookworm-slim
COPY --from=builder /shrink /usr/local/bin/shrink
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["shrink"]
