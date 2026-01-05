# Stage 1: Build the Rust binary
FROM rust:1.80-bullseye AS builder

ARG TARGETOS
ARG TARGETARCH

# Install protobuf compiler
RUN apt-get update && apt-get install -y \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy manifests
COPY Cargo.toml Cargo.lock ./

# Create a dummy main to cache dependencies
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release
RUN rm -rf src

# Copy source code
COPY . .

# Remove the dummy build artifacts and build the real binary
RUN rm -rf target/release/deps/numerix*
RUN cargo build --release

# Stage 2: Minimal distroless runtime
FROM gcr.io/distroless/cc:nonroot

WORKDIR /

COPY --from=builder /app/target/release/numerix .

EXPOSE 8082


ENTRYPOINT ["/numerix"]