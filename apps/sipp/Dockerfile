# Build from repo root: docker build -t sipp -f apps/sipp/Dockerfile .
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

RUN cargo build --release -p sipp-so

# Copy real source
COPY crates/auth/src crates/auth/src
COPY apps/sipp/src apps/sipp/src
COPY apps/sipp/static apps/sipp/static
COPY apps/sipp/templates apps/sipp/templates

RUN touch apps/sipp/src/*.rs crates/auth/src/*.rs && cargo build --release -p sipp-so

FROM debian:bookworm-slim
COPY --from=builder /app/target/release/sipp /usr/local/bin/sipp
WORKDIR /data
EXPOSE 3000
CMD ["sipp", "server", "--port", "3000", "--host", "0.0.0.0"]
