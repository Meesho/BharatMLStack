package com.bharatml.featurestore.connector.horizon;

import java.util.concurrent.atomic.AtomicReference;

/**
 * Thread-safe holder for source mapping response.
 * Each instance maintains its own source mapping, allowing multiple jobs
 * to use different mappings independently.
 */
public class SourceMappingHolder {
    private final AtomicReference<SourceMappingResponse> currentResponse = new AtomicReference<>();

    /**
     * Creates a new SourceMappingHolder instance.
     */
    public SourceMappingHolder() {
    }

    /**
     * Updates the source mapping response.
     *
     * @param newResponse the new source mapping response
     */
    public void update(SourceMappingResponse newResponse) {
        currentResponse.set(newResponse);
    }

    /**
     * Gets the current source mapping response.
     *
     * @return the current source mapping response, or null if not set
     */
    public SourceMappingResponse get() {
        return currentResponse.get();
    }
}
