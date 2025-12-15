package com.bharatml.featurestore.connector.horizon;

import java.util.concurrent.atomic.AtomicReference;

public class SourceMappingHolder {
    private static final AtomicReference<SourceMappingResponse> CURRENT_RESPONSE = new AtomicReference<>();

    public static void update(SourceMappingResponse newResponse) {
        CURRENT_RESPONSE.set(newResponse);
    }
    public static SourceMappingResponse get() {
        return CURRENT_RESPONSE.get();
    }

}
