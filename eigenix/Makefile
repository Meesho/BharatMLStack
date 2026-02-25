# Eigenix KMeans: production build
# Requires: BLAS (OpenBLAS or similar), FAISS, OpenMP
#
# On macOS with Apple Clang: install libomp (brew install libomp) and either
# use a g++ with -fopenmp, or set OPENMP_EXTRA="-Xpreprocessor -fopenmp" and
# link -lomp (e.g. LDFLAGS += -lomp).
#
# Usage:
#   make          # build library + benchmark
#   make run      # run benchmark
#   make clean

CXX = g++
CXXFLAGS = -O3 -std=c++17 -fopenmp -I include $(OPENMP_EXTRA)
LDFLAGS = -fopenmp

# Adjust to your system: often -lopenblas or -lblas, and faiss may need -lgomp -lopenblas
BLAS_LIBS = -lblas
FAISS_LIBS = -lfaiss

EIGENIX_SRC = src/kmeans_blas.cpp src/kmeans_faiss.cpp src/metrics.cpp
EIGENIX_OBJ = $(EIGENIX_SRC:src/%.cpp=build/%.o)

.PHONY: all clean benchmark run

all: build/libeigenix_kmeans.a benchmark/benchmark_kmeans

build:
	mkdir -p build

build/%.o: src/%.cpp | build
	$(CXX) $(CXXFLAGS) -c -o $@ $<

build/libeigenix_kmeans.a: $(EIGENIX_OBJ)
	ar rcs $@ $^
	ranlib $@

benchmark/benchmark_kmeans: benchmark/benchmark_kmeans.cpp build/libeigenix_kmeans.a
	$(CXX) $(CXXFLAGS) -I include -o $@ benchmark/benchmark_kmeans.cpp build/libeigenix_kmeans.a $(BLAS_LIBS) $(FAISS_LIBS) $(LDFLAGS)

run: benchmark/benchmark_kmeans
	./benchmark/benchmark_kmeans

clean:
	rm -rf build
	rm -f benchmark/benchmark_kmeans
