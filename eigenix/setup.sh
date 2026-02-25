#!/usr/bin/env bash
set -euo pipefail

# Eigenix KMeans Benchmark Suite — dependency installer for Ubuntu 22.04+

echo "=== Installing build tools ==="
sudo apt-get update
sudo apt-get install -y \
    build-essential \
    cmake \
    git \
    pkg-config \
    wget \
    libopenblas-dev \
    libgomp1 \
    libomp-dev \
    libgtest-dev \
    googletest

echo "=== Building GTest (if needed) ==="
if ! pkg-config --exists gtest 2>/dev/null; then
    cd /usr/src/googletest
    sudo cmake -B build -DCMAKE_INSTALL_PREFIX=/usr/local
    sudo cmake --build build -j"$(nproc)"
    sudo cmake --install build
    cd -
fi

echo "=== Installing FAISS (CPU-only) ==="
if ! pkg-config --exists faiss 2>/dev/null && [ ! -f /usr/local/lib/libfaiss.a ]; then
    FAISS_VERSION="v1.9.0"
    FAISS_DIR=$(mktemp -d)
    git clone --depth 1 --branch "$FAISS_VERSION" https://github.com/facebookresearch/faiss.git "$FAISS_DIR"
    cd "$FAISS_DIR"
    cmake -B build \
        -DFAISS_ENABLE_GPU=OFF \
        -DFAISS_ENABLE_PYTHON=OFF \
        -DBUILD_TESTING=OFF \
        -DFAISS_OPT_LEVEL=avx2 \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_INSTALL_PREFIX=/usr/local \
        -DBLA_VENDOR=OpenBLAS
    cmake --build build -j"$(nproc)"
    sudo cmake --install build
    cd -
    rm -rf "$FAISS_DIR"
fi

echo "=== Verifying installations ==="
echo -n "  CMake: "; cmake --version | head -1
echo -n "  OpenBLAS: "; dpkg -s libopenblas-dev 2>/dev/null | grep Version || echo "not found"
echo -n "  FAISS: "; ls /usr/local/lib/libfaiss* 2>/dev/null | head -1 || echo "not found"
echo -n "  GTest: "; pkg-config --modversion gtest 2>/dev/null || echo "installed (no pkg-config)"

echo ""
echo "=== Done. Build with: ==="
echo "  cd eigenix && cmake -B build -DCMAKE_BUILD_TYPE=Release && cmake --build build -j\$(nproc)"
