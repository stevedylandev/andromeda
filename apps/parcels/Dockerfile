# Build from repo root: docker build -t parcels -f apps/parcels/Dockerfile .
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

RUN cargo build --release -p parcels

# Copy real source
COPY crates/auth/src crates/auth/src
COPY apps/parcels/src apps/parcels/src
COPY apps/parcels/templates apps/parcels/templates
COPY apps/parcels/static apps/parcels/static

RUN touch apps/parcels/src/*.rs crates/auth/src/*.rs && cargo build --release -p parcels

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/target/release/parcels ./parcels
COPY --from=builder /app/apps/parcels/static ./static
COPY --from=builder /app/apps/parcels/templates ./templates
EXPOSE 3012
CMD ["./parcels"]
