package com.bharatml.featurestore.core;

import org.junit.jupiter.api.Test;
import persist.QueryOuterClass;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for FeatureConverter to verify Protobuf serialization.
 */
public class FeatureConverterTest {

    @Test
    public void testSingleFeatureEventConversion() {
        // Create a sample FeatureEvent
        FeatureEvent event = new FeatureEvent();
        event.setEntityLabel("user");
        event.setKeysSchema(Arrays.asList("user_id"));
        Map<String, Object> keys = new HashMap<>();
        keys.put("user_id", "12345");
        event.setKeys(keys);
        event.setFeatureGroupLabel("user_features");
        event.setFeatureLabels(Arrays.asList("age", "score", "active"));
        Map<String, Object> featureValues = new HashMap<>();
        featureValues.put("age", 30);
        featureValues.put("score", 95.5);
        featureValues.put("active", true);
        event.setFeatureValues(featureValues);

        // Convert to Protobuf
        QueryOuterClass.Query query = FeatureConverter.toProto(event);

        // Verify entity_label
        assertEquals("user", query.getEntityLabel());

        // Verify keys_schema
        assertEquals(1, query.getKeysSchemaCount());
        assertEquals("user_id", query.getKeysSchema(0));

        // Verify feature_group_schema
        assertEquals(1, query.getFeatureGroupSchemaCount());
        QueryOuterClass.FeatureGroupSchema fgSchema = query.getFeatureGroupSchema(0);
        assertEquals("user_features", fgSchema.getLabel());
        assertEquals(3, fgSchema.getFeatureLabelsCount());
        assertEquals("age", fgSchema.getFeatureLabels(0));
        assertEquals("score", fgSchema.getFeatureLabels(1));
        assertEquals("active", fgSchema.getFeatureLabels(2));

        // Verify data
        assertEquals(1, query.getDataCount());
        QueryOuterClass.Data data = query.getData(0);
        assertEquals(1, data.getKeyValuesCount());
        assertEquals("12345", data.getKeyValues(0));

        assertEquals(1, data.getFeatureValuesCount());
        QueryOuterClass.FeatureValues fv = data.getFeatureValues(0);
        QueryOuterClass.Values values = fv.getValues();
        
        // Verify feature values (age=30 as int32, score=95.5 as fp64, active=true as bool)
        assertEquals(1, values.getInt32ValuesCount());
        assertEquals(30, values.getInt32Values(0));
        
        assertEquals(1, values.getFp64ValuesCount());
        assertEquals(95.5, values.getFp64Values(0), 0.001);
        
        assertEquals(1, values.getBoolValuesCount());
        assertTrue(values.getBoolValues(0));
    }

    @Test
    public void testProtobufBytesSerialization() {
        FeatureEvent event = new FeatureEvent();
        event.setEntityLabel("test_entity");
        event.setKeysSchema(Arrays.asList("id"));
        Map<String, Object> keys = new HashMap<>();
        keys.put("id", "test-id");
        event.setKeys(keys);
        event.setFeatureGroupLabel("test_features");
        event.setFeatureLabels(Arrays.asList("value"));
        Map<String, Object> featureValues = new HashMap<>();
        featureValues.put("value", "test-value");
        event.setFeatureValues(featureValues);

        // Convert to Protobuf
        QueryOuterClass.Query query = FeatureConverter.toProto(event);

        // Serialize to bytes
        byte[] protobufBytes = query.toByteArray();
        assertNotNull(protobufBytes);
        assertTrue(protobufBytes.length > 0);

        // Deserialize back and verify
        try {
            QueryOuterClass.Query deserializedQuery = QueryOuterClass.Query.parseFrom(protobufBytes);
            assertEquals("test_entity", deserializedQuery.getEntityLabel());
            assertEquals("test-id", deserializedQuery.getData(0).getKeyValues(0));
            assertEquals("test-value", deserializedQuery.getData(0).getFeatureValues(0)
                    .getValues().getStringValues(0));
        } catch (Exception e) {
            fail("Failed to deserialize Protobuf bytes: " + e.getMessage());
        }
    }

    @Test
    public void testNullFeatureEventThrowsException() {
        assertThrows(IllegalArgumentException.class, () -> {
            FeatureConverter.toProto((FeatureEvent) null);
        });
    }
}

