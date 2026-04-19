# Build from repo root: docker build -t shrink -f apps/shrink/Dockerfile .
FROM lukemathwalker/cargo-chef:latest-rust-1-slim-bookworm AS chef
WORKDIR /app

FROM chef AS planner
COPY . .
RUN cargo chef prepare --recipe-path recipe.json

FROM chef AS builder
RUN apt-get update && apt-get install -y pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*
COPY --from=planner /app/recipe.json recipe.json
RUN cargo chef cook --release --recipe-path recipe.json -p shrink
COPY . .
RUN cargo build --release -p shrink

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/target/release/shrink /usr/local/bin/shrink
COPY --from=builder /app/apps/shrink/static /app/static
EXPOSE 3000
ENV HOST=0.0.0.0
CMD ["shrink"]
