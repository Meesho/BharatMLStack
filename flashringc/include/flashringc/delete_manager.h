#pragma once

#include <cstdint>

class KeyIndex;
class RingDevice;

class DeleteManager {
public:
    struct Config {
        double eviction_threshold = 0.95;
        double clear_threshold = 0.85;
        int base_k = 4;
        int max_discards_per_put = 2;
    };

    DeleteManager(const Config& cfg, KeyIndex& index, RingDevice& device);

    void on_put();
    void on_tick();

    bool has_backlog() const;
    bool is_discarded(uint32_t mem_id) const;
    uint32_t discarded_through_mem_id() const;
    bool is_evicting() const;

    void reset();

private:
    void maybe_discard_blocks();
    void discard_oldest_block();
    void amortize_index_cleanup(int effective_k);
    void advance_to_next_memid();
    bool needs_discard() const;
    uint64_t memid_to_block_offset(uint32_t mem_id) const;

    Config cfg_;

    bool     evicting_                = false;
    uint32_t oldest_mem_id_           = 0;
    uint32_t discarded_through_mem_id_ = 0;
    bool     has_discarded_           = false;

    uint32_t cleaning_mem_id_         = 0;
    uint32_t cleaning_cursor_         = 0;
    bool     cleaning_in_progress_    = false;

    KeyIndex&   index_;
    RingDevice& device_;
};
