package com.bharatml.featurestore.connector;

import com.bharatml.featurestore.core.FeatureEvent;
import org.apache.flink.api.common.serialization.SerializationSchema;

import java.nio.charset.StandardCharsets;

/**
 * Builds a key (byte[]) from FeatureEvent. Uses entityLabel as key (fallback to "default").
 */
public class FeatureEventKeySerialization implements SerializationSchema<FeatureEvent> {

    @Override
    public byte[] serialize(FeatureEvent element) {
        String key = element.getEntityLabel() != null ? element.getEntityLabel() : "default";
        return key.getBytes(StandardCharsets.UTF_8);
    }
}
