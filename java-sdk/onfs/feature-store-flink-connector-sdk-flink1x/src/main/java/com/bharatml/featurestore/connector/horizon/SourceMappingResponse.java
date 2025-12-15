package com.bharatml.featurestore.connector.horizon;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;
import java.util.Map;

/**
 * Response structure from Horizon /get-source-mapping endpoint.
 * Represents the complete metadata mapping for features.
 */
public class SourceMappingResponse {
    
    @JsonProperty("data")
    private List<StorageProviderData> data;
    
    @JsonProperty("keys")
    private Map<String, List<String>> keys;
    
    public SourceMappingResponse() {
    }
    
    public List<StorageProviderData> getData() {
        return data;
    }
    
    public Map<String, List<String>> getKeys() {
        return keys;
    }
    
    /**
     * Checks if the given entity label exists in the keys mapping.
     */
    public boolean hasEntityLabel(String entityLabel) {
        return keys != null && keys.containsKey(entityLabel);
    }
    
    /**
     * Checks if the given entity, feature group, and feature labels are registered.
     */
    public boolean hasFeatureMapping(String entityLabel, String featureGroupLabel, List<String> featureLabels) {
        if (data == null) {
            return false;
        }
        
        for (StorageProviderData storageData : data) {
            if (storageData.hasFeatureMapping(entityLabel, featureGroupLabel, featureLabels)) {
                return true;
            }
        }
        
        return false;
    }
}

