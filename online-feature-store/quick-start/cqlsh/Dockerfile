FROM python:3.11-slim

# Install required system packages (includes git now!)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git gcc g++ libffi-dev libssl-dev libpython3-dev \
    && rm -rf /var/lib/apt/lists/*

# Install cqlsh client from GitHub
RUN pip install git+https://github.com/scylladb/scylla-cqlsh.git

# Set default entrypoint
ENTRYPOINT ["cqlsh"]