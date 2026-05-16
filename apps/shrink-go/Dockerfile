# Build from repo root: docker build -t shrink-go -f apps/shrink-go/Dockerfile .
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/shrink-go/go.mod apps/shrink-go/go.sum ./apps/shrink-go/
WORKDIR /app/apps/shrink-go
RUN go mod download
COPY apps/shrink-go/ ./
RUN CGO_ENABLED=0 go build -o /shrink-go .

FROM debian:bookworm-slim
COPY --from=builder /shrink-go /usr/local/bin/shrink-go
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["shrink-go"]
