#!/usr/bin/env bash
set -euo pipefail

OS="$(uname -s)"
ARCH="$(uname -m)"
NPROC="$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)"

echo "=== Eigenix KMeans Benchmark — Dependency Installer ==="
echo "    OS=$OS  ARCH=$ARCH  NPROC=$NPROC"
echo ""

# ---------------------------------------------------------------------------
# macOS (Homebrew)
# ---------------------------------------------------------------------------
if [ "$OS" = "Darwin" ]; then
    if ! command -v brew &>/dev/null; then
        echo "ERROR: Homebrew not found. Install from https://brew.sh" >&2
        exit 1
    fi

    echo "=== Installing build tools (Homebrew) ==="
    brew install cmake openblas libomp googletest pkg-config 2>/dev/null || true

    BREW_PREFIX="$(brew --prefix)"

    echo "=== Installing FAISS (CPU-only, from source) ==="
    if [ ! -f "$BREW_PREFIX/lib/libfaiss.a" ] && [ ! -f /usr/local/lib/libfaiss.a ]; then
        FAISS_VERSION="v1.9.0"
        FAISS_DIR=$(mktemp -d)
        git clone --depth 1 --branch "$FAISS_VERSION" \
            https://github.com/facebookresearch/faiss.git "$FAISS_DIR"
        cd "$FAISS_DIR"

        OPT_LEVEL="generic"
        if [ "$ARCH" = "x86_64" ]; then
            OPT_LEVEL="avx2"
        fi

        cmake -B build \
            -DFAISS_ENABLE_GPU=OFF \
            -DFAISS_ENABLE_PYTHON=OFF \
            -DBUILD_TESTING=OFF \
            -DFAISS_OPT_LEVEL="$OPT_LEVEL" \
            -DCMAKE_BUILD_TYPE=Release \
            -DCMAKE_INSTALL_PREFIX="$BREW_PREFIX" \
            -DCMAKE_PREFIX_PATH="$BREW_PREFIX/opt/openblas;$BREW_PREFIX/opt/libomp" \
            -DOpenMP_C_FLAGS="-Xpreprocessor -fopenmp -I$BREW_PREFIX/opt/libomp/include" \
            -DOpenMP_CXX_FLAGS="-Xpreprocessor -fopenmp -I$BREW_PREFIX/opt/libomp/include" \
            -DOpenMP_C_LIB_NAMES="omp" \
            -DOpenMP_CXX_LIB_NAMES="omp" \
            -DOpenMP_omp_LIBRARY="$BREW_PREFIX/opt/libomp/lib/libomp.dylib"
        cmake --build build -j"$NPROC"
        cmake --install build
        cd -
        rm -rf "$FAISS_DIR"
    else
        echo "  FAISS already installed, skipping."
    fi

    echo ""
    echo "=== Verifying installations ==="
    echo -n "  CMake: "; cmake --version | head -1
    echo -n "  OpenBLAS: "; brew list --versions openblas 2>/dev/null || echo "not found"
    echo -n "  libomp: "; brew list --versions libomp 2>/dev/null || echo "not found"
    echo -n "  FAISS: "; ls "$BREW_PREFIX/lib/libfaiss"* 2>/dev/null | head -1 || echo "not found"
    echo -n "  GTest: "; brew list --versions googletest 2>/dev/null || echo "not found"

# ---------------------------------------------------------------------------
# Linux (apt-get)
# ---------------------------------------------------------------------------
elif [ "$OS" = "Linux" ]; then
    echo "=== Installing build tools (apt-get) ==="
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
        sudo cmake --build build -j"$NPROC"
        sudo cmake --install build
        cd -
    fi

    echo "=== Installing FAISS (CPU-only, from source) ==="
    if ! pkg-config --exists faiss 2>/dev/null && [ ! -f /usr/local/lib/libfaiss.a ]; then
        FAISS_VERSION="v1.9.0"
        FAISS_DIR=$(mktemp -d)
        git clone --depth 1 --branch "$FAISS_VERSION" \
            https://github.com/facebookresearch/faiss.git "$FAISS_DIR"
        cd "$FAISS_DIR"

        OPT_LEVEL="generic"
        if [ "$ARCH" = "x86_64" ]; then
            OPT_LEVEL="avx2"
        fi

        cmake -B build \
            -DFAISS_ENABLE_GPU=OFF \
            -DFAISS_ENABLE_PYTHON=OFF \
            -DBUILD_TESTING=OFF \
            -DFAISS_OPT_LEVEL="$OPT_LEVEL" \
            -DCMAKE_BUILD_TYPE=Release \
            -DCMAKE_INSTALL_PREFIX=/usr/local \
            -DBLA_VENDOR=OpenBLAS
        cmake --build build -j"$NPROC"
        sudo cmake --install build
        cd -
        rm -rf "$FAISS_DIR"
    else
        echo "  FAISS already installed, skipping."
    fi

    echo ""
    echo "=== Verifying installations ==="
    echo -n "  CMake: "; cmake --version | head -1
    echo -n "  OpenBLAS: "; dpkg -s libopenblas-dev 2>/dev/null | grep Version || echo "not found"
    echo -n "  FAISS: "; ls /usr/local/lib/libfaiss* 2>/dev/null | head -1 || echo "not found"
    echo -n "  GTest: "; pkg-config --modversion gtest 2>/dev/null || echo "installed (no pkg-config)"

else
    echo "ERROR: Unsupported OS '$OS'. Only Linux and macOS are supported." >&2
    exit 1
fi

echo ""
echo "=== Done. Build with: ==="
echo "  cd eigenix && cmake -B build -DCMAKE_BUILD_TYPE=Release && cmake --build build -j$NPROC"
