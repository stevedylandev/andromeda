# Build from repo root: docker build -t og -f apps/og/Dockerfile .
FROM rust:1-slim-bookworm AS builder
WORKDIR /app

# Copy workspace manifests
COPY Cargo.toml Cargo.lock ./
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

RUN cargo build --release -p og

# Copy real source
COPY apps/og/src apps/og/src
COPY apps/og/templates apps/og/templates
COPY apps/og/static apps/og/static

RUN touch apps/og/src/*.rs && cargo build --release -p og

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/target/release/og .
EXPOSE 3000
CMD ["./og"]
