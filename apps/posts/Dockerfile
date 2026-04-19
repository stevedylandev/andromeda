# Build from repo root: docker build -t posts -f apps/posts/Dockerfile .
FROM rust:1-slim-bookworm AS builder
RUN apt-get update && apt-get install -y pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app

# Copy workspace manifests
COPY Cargo.toml Cargo.lock .
COPY crates/auth/Cargo.toml crates/auth/
COPY crates/db/Cargo.toml crates/db/
COPY crates/darkmatter-css/Cargo.toml crates/darkmatter-css/
COPY apps/sipp/Cargo.toml apps/sipp/
COPY apps/feeds/Cargo.toml apps/feeds/
COPY apps/parcels/Cargo.toml apps/parcels/
COPY apps/jotts/Cargo.toml apps/jotts/
COPY apps/og/Cargo.toml apps/og/
COPY apps/shrink/Cargo.toml apps/shrink/
COPY apps/cellar/Cargo.toml apps/cellar/
COPY apps/posts/Cargo.toml apps/posts/

# Create stubs for dependency caching
RUN mkdir -p crates/auth/src && echo '' > crates/auth/src/lib.rs \
    && mkdir -p crates/db/src && echo '' > crates/db/src/lib.rs \
    && mkdir -p crates/darkmatter-css/src && echo '' > crates/darkmatter-css/src/lib.rs \
    && mkdir -p crates/darkmatter-css/assets \
    && for app in sipp feeds parcels jotts og shrink cellar posts; do \
         mkdir -p apps/$app/src && echo 'fn main() {}' > apps/$app/src/main.rs; \
       done

RUN cargo build --release -p posts

# Copy real source
COPY crates/auth/src crates/auth/src
COPY crates/db/src crates/db/src
COPY crates/darkmatter-css/src crates/darkmatter-css/src
COPY crates/darkmatter-css/assets crates/darkmatter-css/assets
COPY apps/posts/src apps/posts/src
COPY apps/posts/static apps/posts/static
COPY apps/posts/templates apps/posts/templates

RUN touch apps/posts/src/*.rs crates/auth/src/*.rs crates/db/src/*.rs crates/darkmatter-css/src/*.rs && cargo build --release -p posts

FROM debian:bookworm-slim
COPY --from=builder /app/target/release/posts /usr/local/bin/posts
WORKDIR /data
EXPOSE 3000
ENV HOST=0.0.0.0
ENV PORT=3000
CMD ["posts"]
