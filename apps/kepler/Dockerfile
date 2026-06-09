# Build from repo root: docker build -t kepler -f apps/kepler/Dockerfile .
FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY apps/kepler/go.mod apps/kepler/go.sum ./apps/kepler/
WORKDIR /app/apps/kepler
RUN go mod download
COPY apps/kepler/ ./
RUN CGO_ENABLED=0 go build -o /kepler .

FROM debian:bookworm-slim
COPY --from=builder /kepler /usr/local/bin/kepler
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=4747
ENV KEPLER_REPO_ROOT=/data/repos
EXPOSE 4747
CMD ["kepler"]
