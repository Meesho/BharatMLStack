package com.bharatml.featurestore.core;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * POJO representing a feature event to be serialized to Protobuf.
 */
public class FeatureEvent {
    private String entityLabel;
    private List<String> keysSchema;
    private Map<String, Object> keys;
    private String featureGroupLabel;
    private List<String> featureLabels;
    private Map<String, Object> featureValues;
    private Long timestamp;

    public FeatureEvent() {
        this.keysSchema = new ArrayList<>();
        this.keys = new HashMap<>();
        this.featureLabels = new ArrayList<>();
        this.featureValues = new HashMap<>();
    }

    public FeatureEvent(String entityLabel, List<String> keysSchema, Map<String, Object> keys,
                        String featureGroupLabel, List<String> featureLabels, Map<String, Object> featureValues) {
        this.entityLabel = entityLabel;
        this.keysSchema = keysSchema != null ? keysSchema : new ArrayList<>();
        this.keys = keys != null ? keys : new HashMap<>();
        this.featureGroupLabel = featureGroupLabel;
        this.featureLabels = featureLabels != null ? featureLabels : new ArrayList<>();
        this.featureValues = featureValues != null ? featureValues : new HashMap<>();
    }

    public String getEntityLabel() {
        return entityLabel;
    }

    public void setEntityLabel(String entityLabel) {
        this.entityLabel = entityLabel;
    }

    public List<String> getKeysSchema() {
        return keysSchema;
    }

    public void setKeysSchema(List<String> keysSchema) {
        this.keysSchema = keysSchema;
    }

    public Map<String, Object> getKeys() {
        return keys;
    }

    public void setKeys(Map<String, Object> keys) {
        this.keys = keys;
    }

    public String getFeatureGroupLabel() {
        return featureGroupLabel;
    }

    public void setFeatureGroupLabel(String featureGroupLabel) {
        this.featureGroupLabel = featureGroupLabel;
    }

    public List<String> getFeatureLabels() {
        return featureLabels;
    }

    public void setFeatureLabels(List<String> featureLabels) {
        this.featureLabels = featureLabels;
    }

    public Map<String, Object> getFeatureValues() {
        return featureValues;
    }

    public void setFeatureValues(Map<String, Object> featureValues) {
        this.featureValues = featureValues;
    }

    public Long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(Long timestamp) {
        this.timestamp = timestamp;
    }

    @Override
    public String toString() {
        return "FeatureEvent{" +
                "entityLabel='" + entityLabel + '\'' +
                ", keysSchema=" + keysSchema +
                ", keys=" + keys +
                ", featureGroupLabel='" + featureGroupLabel + '\'' +
                ", featureLabels=" + featureLabels +
                ", featureValues=" + featureValues +
                ", timestamp=" + timestamp +
                '}';
    }
}

