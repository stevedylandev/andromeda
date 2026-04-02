# Build from repo root: docker build -t shrink -f apps/shrink/Dockerfile .
FROM rust:1-slim-bookworm AS builder
RUN apt-get update && apt-get install -y pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app

# Copy workspace manifests
COPY Cargo.toml Cargo.lock .
COPY crates/auth/Cargo.toml crates/auth/
COPY apps/sipp/Cargo.toml apps/sipp/
COPY apps/feeds/Cargo.toml apps/feeds/
COPY apps/parcels/Cargo.toml apps/parcels/
COPY apps/jotts/Cargo.toml apps/jotts/
COPY apps/og/Cargo.toml apps/og/
COPY apps/shrink/Cargo.toml apps/shrink/

# Create stubs for dependency caching
RUN mkdir -p crates/auth/src && echo '' > crates/auth/src/lib.rs \
    && for app in sipp feeds parcels jotts og shrink; do \
         mkdir -p apps/$app/src && echo 'fn main() {}' > apps/$app/src/main.rs; \
       done

RUN cargo build --release -p shrink

# Copy real source
COPY apps/shrink/src apps/shrink/src
COPY apps/shrink/templates apps/shrink/templates
COPY apps/shrink/static apps/shrink/static

RUN touch apps/shrink/src/*.rs && cargo build --release -p shrink

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/target/release/shrink /usr/local/bin/shrink
COPY --from=builder /app/apps/shrink/static /app/static
EXPOSE 3000
ENV HOST=0.0.0.0
CMD ["shrink"]
