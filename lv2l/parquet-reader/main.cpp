#include <arrow/api.h>
#include <arrow/io/api.h>
#include <arrow/pretty_print.h>
#include <parquet/arrow/reader.h>

#include <filesystem>
#include <iostream>
#include <optional>
#include <string>
#include <vector>

namespace fs = std::filesystem;

static std::vector<fs::path> find_all_snappy_parquet_in(const fs::path& data_dir) {
    std::vector<fs::path> snappy_files;
    
    if (!fs::exists(data_dir) || !fs::is_directory(data_dir)) return snappy_files;

    for (auto& p : fs::directory_iterator(data_dir)) {
        if (!p.is_regular_file()) continue;
        if (p.path().extension() == ".parquet") {
            const auto fname = p.path().filename().string();
            // Only collect *.snappy.parquet files
            if (fname.size() >= 16 && fname.rfind(".snappy.parquet") == fname.size() - 16) {
                snappy_files.push_back(p.path());
            }
        }
    }
    
    return snappy_files;
}

static void print_head(const std::shared_ptr<arrow::Table>& table, int64_t n_rows) {
    for (const auto& row : table->rows(n_rows)) {
        std::cout << row << "\n";
    }
    auto sliced = table->Slice(0, n_rows);
    arrow::PrettyPrintOptions opts(/*indent=*/0);
    opts.window = n_rows;
    auto st = arrow::PrettyPrint(*sliced, opts, &std::cout);
    if (!st.ok()) {
        std::cerr << "PrettyPrint failed: " << st.ToString() << "\n";
    }
}

int main(int argc, char** argv) {
    fs::path data_dir = (argc > 1) ? fs::path(argv[1]) : fs::path("data");

    auto snappy_files = find_all_snappy_parquet_in(data_dir);
    if (snappy_files.empty()) {
        std::cerr << "No .snappy.parquet files found in: " << data_dir << "\n";
        return 1;
    }
    
    std::cout << "Found " << snappy_files.size() << " .snappy.parquet file(s):\n";
    for (const auto& file_path : snappy_files) {
        std::cout << "  " << file_path << "\n";
    }
    
    // Process the first file for demonstration
    auto file_path = snappy_files[0];
    std::cout << "\nReading first Parquet file: " << file_path << "\n";

    // Open file (Result<T> API)
    auto infile_res = arrow::io::ReadableFile::Open(file_path.string());
    if (!infile_res.ok()) {
        std::cerr << "Failed to open file: " << infile_res.status().ToString() << "\n";
        return 1;
    }
    auto infile = std::move(infile_res).ValueOrDie();

    // Create parquet::arrow::FileReader (Result<unique_ptr<FileReader>> API)
    auto reader_res = parquet::arrow::OpenFile(infile, arrow::default_memory_pool());
    if (!reader_res.ok()) {
        std::cerr << "Failed to create Parquet reader: " << reader_res.status().ToString() << "\n";
        return 1;
    }
    std::unique_ptr<parquet::arrow::FileReader> pq_reader = std::move(reader_res).ValueOrDie();

    // Read whole file to Arrow Table (pointer out-param API)
    std::shared_ptr<arrow::Table> table;
    auto st2 = pq_reader->ReadTable(&table);
    if (!st2.ok()) {
        std::cerr << "Failed to read Parquet as Arrow Table: " << st2.ToString() << "\n";
        return 1;
    }

    // Log schema + a few rows
    std::cout << "Schema:\n" << table->schema()->ToString(/*show_metadata=*/true) << "\n";
    std::cout << "Rows: " << table->num_rows()
              << ", Columns: " << table->num_columns() << "\n";

    std::cout << "\nFirst 10 rows:\n";
    print_head(table, /*n_rows=*/10);

    return 0;
}
