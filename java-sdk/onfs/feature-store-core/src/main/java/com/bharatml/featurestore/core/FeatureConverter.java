package com.bharatml.featurestore.core;

import persist.QueryOuterClass;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Converts FeatureEvent POJO to Protobuf Query message.
 * Strictly Protobuf-only serialization (no JSON fallback).
 */
public class FeatureConverter {

    /**
     * Converts a single FeatureEvent to a Protobuf Query message.
     * 
     * @param ev FeatureEvent to convert
     * @return Protobuf Query message
     */
    public static persist.QueryOuterClass.Query toProto(FeatureEvent ev) {
        if (ev == null) {
            throw new IllegalArgumentException("FeatureEvent cannot be null");
        }

        QueryOuterClass.Query.Builder queryBuilder = QueryOuterClass.Query.newBuilder();

        // Set entity_label
        if (ev.getEntityLabel() != null) {
            queryBuilder.setEntityLabel(ev.getEntityLabel());
        }

        // Set keys_schema
        if (ev.getKeysSchema() != null) {
            queryBuilder.addAllKeysSchema(ev.getKeysSchema());
        }

        // Build FeatureGroupSchema
        if (ev.getFeatureGroupLabel() != null || 
            (ev.getFeatureLabels() != null && !ev.getFeatureLabels().isEmpty())) {
            QueryOuterClass.FeatureGroupSchema.Builder fgSchemaBuilder = 
                QueryOuterClass.FeatureGroupSchema.newBuilder();
            
            if (ev.getFeatureGroupLabel() != null) {
                fgSchemaBuilder.setLabel(ev.getFeatureGroupLabel());
            }
            
            if (ev.getFeatureLabels() != null) {
                fgSchemaBuilder.addAllFeatureLabels(ev.getFeatureLabels());
            }
            
            queryBuilder.addFeatureGroupSchema(fgSchemaBuilder.build());
        }

        // Build Data
        QueryOuterClass.Data.Builder dataBuilder = QueryOuterClass.Data.newBuilder();

        // Add key_values in the order of keys_schema
        if (ev.getKeysSchema() != null && ev.getKeys() != null) {
            for (String keyName : ev.getKeysSchema()) {
                Object value = ev.getKeys().get(keyName);
                if (value != null) {
                    dataBuilder.addKeyValues(String.valueOf(value));
                } else {
                    dataBuilder.addKeyValues("");
                }
            }
        }

        // Build FeatureValues
        QueryOuterClass.FeatureValues.Builder featureValuesBuilder = 
            QueryOuterClass.FeatureValues.newBuilder();
        QueryOuterClass.Values.Builder valuesBuilder = QueryOuterClass.Values.newBuilder();

        if (ev.getFeatureLabels() != null && ev.getFeatureValues() != null) {
            for (String featureLabel : ev.getFeatureLabels()) {
                Object value = ev.getFeatureValues().get(featureLabel);
                addValueToBuilder(valuesBuilder, value);
            }
        }

        featureValuesBuilder.setValues(valuesBuilder.build());
        dataBuilder.addFeatureValues(featureValuesBuilder.build());

        queryBuilder.addData(dataBuilder.build());

        return queryBuilder.build();
    }

    /**
     * Adds a value to the Values builder based on its type.
     */
    private static void addValueToBuilder(QueryOuterClass.Values.Builder builder, Object value) {
        if (value == null) {
            return;
        }

        if (value instanceof Float || value instanceof Double) {
            double doubleValue = ((Number) value).doubleValue();
            builder.addFp64Values(doubleValue);
        } else if (value instanceof Integer) {
            builder.addInt32Values((Integer) value);
        } else if (value instanceof Long) {
            builder.addInt64Values((Long) value);
        } else if (value instanceof String) {
            builder.addStringValues((String) value);
        } else if (value instanceof Boolean) {
            builder.addBoolValues((Boolean) value);
        } else if (value instanceof List) {
            // Handle list values - convert to appropriate type
            List<?> list = (List<?>) value;
            if (!list.isEmpty()) {
                Object first = list.get(0);
                if (first instanceof Number) {
                    for (Object item : list) {
                        double doubleValue = ((Number) item).doubleValue();
                        builder.addFp64Values(doubleValue);
                    }
                } else if (first instanceof String) {
                    for (Object item : list) {
                        builder.addStringValues((String) item);
                    }
                } else if (first instanceof Boolean) {
                    for (Object item : list) {
                        builder.addBoolValues((Boolean) item);
                    }
                }
            }
        } else {
            // Fallback to string representation
            builder.addStringValues(String.valueOf(value));
        }
    }
}

