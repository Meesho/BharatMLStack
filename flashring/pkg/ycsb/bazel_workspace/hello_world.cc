#include <iostream>
#include <string>
#include <chrono>
#include <random>

#include "absl/container/flat_hash_map.h"

constexpr int kNumElements = 1'000'000;

int main() {
  absl::flat_hash_map<int, int> map;
  map.reserve(kNumElements);


  // Random number generator
  std::mt19937 rng(42);
  std::uniform_int_distribution<int> dist(1, kNumElements * 10);

  std::vector<int> keys;
  keys.reserve(kNumElements);
  for (int i = 0; i < kNumElements; ++i) {
    keys.push_back(dist(rng));
  }

  // Insertion benchmark
  auto start_insert = std::chrono::high_resolution_clock::now();
  for (int i = 0; i < kNumElements; ++i) {
    map[keys[i]] = i;
  }
  auto end_insert = std::chrono::high_resolution_clock::now();
  std::chrono::duration<double> insert_duration = end_insert - start_insert;
  std::cout << "Insertion of " << kNumElements << " items took: " << insert_duration.count() << " seconds\n";

  // Lookup benchmark
  auto start_lookup = std::chrono::high_resolution_clock::now();
  size_t found = 0;
  for (int i = 0; i < kNumElements; ++i) {
    if (map.find(keys[i]) != map.end()) {
      ++found;
    }
  }
  auto end_lookup = std::chrono::high_resolution_clock::now();
  std::chrono::duration<double> lookup_duration = end_lookup - start_lookup;
  std::cout << "Lookup of " << kNumElements << " items took: " << lookup_duration.count() << " seconds. Found: " << found << "\n";

  // Optional: Deletion benchmark
  auto start_erase = std::chrono::high_resolution_clock::now();
  for (int i = 0; i < kNumElements; ++i) {
    map.erase(keys[i]);
  }
  auto end_erase = std::chrono::high_resolution_clock::now();
  std::chrono::duration<double> erase_duration = end_erase - start_erase;
  std::cout << "Deletion of " << kNumElements << " items took: " << erase_duration.count() << " seconds\n";

  return 0;
}
