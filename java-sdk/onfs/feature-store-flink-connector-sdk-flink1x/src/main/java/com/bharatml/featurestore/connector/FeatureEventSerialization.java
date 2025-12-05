package com.bharatml.featurestore.connector;

import com.bharatml.featurestore.core.FeatureEvent;
import com.bharatml.featurestore.core.FeatureConverter;
import org.apache.flink.api.common.serialization.SerializationSchema;

/**
 * Serializes FeatureEvent to protobuf bytes using FeatureConverter.
 */
public class FeatureEventSerialization implements SerializationSchema<FeatureEvent> {

    @Override
    public byte[] serialize(FeatureEvent element) {
        // Convert FeatureEvent to protobuf Query and return bytes
        return FeatureConverter.toProto(element).toByteArray();
    }
}
