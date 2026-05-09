# Build from repo root: docker build -t easel -f apps/easel/Dockerfile .
FROM lukemathwalker/cargo-chef:latest-rust-1-slim-bookworm AS chef
WORKDIR /app

FROM chef AS planner
COPY . .
RUN cargo chef prepare --recipe-path recipe.json

FROM chef AS builder
RUN apt-get update && apt-get install -y pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*
COPY --from=planner /app/recipe.json recipe.json
RUN cargo chef cook --release --recipe-path recipe.json -p easel
COPY . .
RUN cargo build --release -p easel

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/easel /usr/local/bin/easel
WORKDIR /data
EXPOSE 3000
ENV HOST=0.0.0.0
ENV PORT=3000
ENV EASEL_DB_PATH=/data/easel.sqlite
CMD ["easel"]
