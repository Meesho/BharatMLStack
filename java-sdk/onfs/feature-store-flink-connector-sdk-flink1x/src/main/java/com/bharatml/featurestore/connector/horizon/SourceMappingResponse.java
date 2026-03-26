package com.bharatml.featurestore.connector.horizon;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.*;

/**
 * Response structure from Horizon /get-source-mapping endpoint.
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
     * Checks if the given entity, feature group, and feature labels are registered.
     */
    public boolean hasFeatureMapping(String entityLabel, String featureGroupLabel, List<String> featureLabels) {
        if (data == null) {
            return false;
        }

        int featureLabelsCount = 0;
        for (String featureLabel : featureLabels) {
            for (StorageProviderData storageData : data) {
                if (storageData.hasFeatureMapping(entityLabel, featureGroupLabel, featureLabel)) {
                    featureLabelsCount++;
                }
            }
        }
        return featureLabelsCount==featureLabels.size();
    }

    public static class StorageProviderData {

        @JsonProperty("storage-provider")
        private String storageProvider;

        @JsonProperty("base-path")
        private List<BasePathData> basePath;

        public StorageProviderData() {
        }

        public String getStorageProvider() {
            return storageProvider;
        }

        public List<BasePathData> getBasePath() {
            return basePath;
        }

        public boolean hasFeatureMapping(String entityLabel, String featureGroupLabel, String featureLabel) {
            if (basePath == null || featureLabel.isEmpty()) {
                return false;
            }
            // Iterate through all base paths
            for (BasePathData basePathData : basePath) {
                for (BasePathData.DataPath dataPath : basePathData.getDataPaths()) {
                    if (dataPath.getEntityLabel() != null && dataPath.getEntityLabel().equals(entityLabel) &&
                        dataPath.getFeatureGroupLabel() != null && dataPath.getFeatureGroupLabel().equals(featureGroupLabel) &&
                        dataPath.getFeatureLabel() != null && dataPath.getFeatureLabel().equals(featureLabel)) {
                        return true;
                    }
                }
            }
            return false;
        }
    }

}

