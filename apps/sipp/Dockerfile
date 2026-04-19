# Build from repo root: docker build -t sipp -f apps/sipp/Dockerfile .
FROM lukemathwalker/cargo-chef:latest-rust-1-slim-bookworm AS chef
WORKDIR /app

FROM chef AS planner
COPY . .
RUN cargo chef prepare --recipe-path recipe.json

FROM chef AS builder
RUN apt-get update && apt-get install -y pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*
COPY --from=planner /app/recipe.json recipe.json
RUN cargo chef cook --release --recipe-path recipe.json -p sipp-so
COPY . .
RUN cargo build --release -p sipp-so

FROM debian:bookworm-slim
COPY --from=builder /app/target/release/sipp /usr/local/bin/sipp
WORKDIR /data
EXPOSE 3000
CMD ["sipp", "server", "--port", "3000", "--host", "0.0.0.0"]
