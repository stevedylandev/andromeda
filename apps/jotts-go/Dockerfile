FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY apps/jotts-go/go.mod apps/jotts-go/go.sum ./
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
