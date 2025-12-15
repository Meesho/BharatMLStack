package com.bharatml.featurestore.connector.horizon;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

/**
 * Represents a storage provider configuration with its base paths.
 */
public class StorageProviderData {
    
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
    
    /**
     * Checks if this storage provider contains the given feature mapping.
     */
    public boolean hasFeatureMapping(String entityLabel, String featureGroupLabel, List<String> featureLabels) {
        if (basePath == null) {
            return false;
        }
        
        for (BasePathData basePathData : basePath) {
            if (basePathData.hasFeatureMapping(entityLabel, featureGroupLabel, featureLabels)) {
                return true;
            }
        }
        
        return false;
    }
}

