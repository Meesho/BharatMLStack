FROM python:3.11-slim

LABEL description="BharatML Stack Database Initialization Container"
LABEL version="1.0"
LABEL purpose="Initializes ScyllaDB, MySQL, etcd for BharatML Stack"

# Install required system packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    # ScyllaDB client dependencies
    git gcc g++ libffi-dev libssl-dev libpython3-dev \
    # MySQL client
    default-mysql-client \
    # Redis client
    redis-tools \
    # General utilities
    curl netcat-openbsd \
    && rm -rf /var/lib/apt/lists/*

# Install ScyllaDB CQL shell
RUN pip install --no-cache-dir git+https://github.com/scylladb/scylla-cqlsh.git

# Install etcdctl
RUN curl -L https://github.com/etcd-io/etcd/releases/download/v3.5.12/etcd-v3.5.12-linux-amd64.tar.gz | \
    tar xz --strip-components=1 -C /usr/local/bin etcd-v3.5.12-linux-amd64/etcdctl

# Create initialization scripts directory
WORKDIR /init-scripts

# Copy initialization scripts
COPY scripts/ ./

# Make scripts executable
RUN chmod +x *.sh

# Default entrypoint runs the main initialization
ENTRYPOINT ["./main-init.sh"]