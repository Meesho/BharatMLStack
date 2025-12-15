package com.bharatml.featurestore.connector.horizon;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

/**
 * Represents a base path (source location) with its data paths (feature mappings).
 */
public class BasePathData {
    
    @JsonProperty("source-base-path")
    private String sourceBasePath;
    
    @JsonProperty("data-paths")
    private List<DataPath> dataPaths;
    
    public BasePathData() {
    }
    
    public String getSourceBasePath() {
        return sourceBasePath;
    }
    
    public List<DataPath> getDataPaths() {
        return dataPaths;
    }
    
    /**
     * Checks if this base path contains the given feature mapping.
     */
    public boolean hasFeatureMapping(String entityLabel, String featureGroupLabel, List<String> featureLabels) {
        if (dataPaths == null) {
            return false;
        }
        
        // Check if all requested features exist in this base path
        for (String featureLabel : featureLabels) {
            for (DataPath dataPath : dataPaths) {
                if (dataPath.getEntityLabel().equals(entityLabel) &&
                    dataPath.getFeatureGroupLabel().equals(featureGroupLabel) &&
                    dataPath.getFeatureLabel().equals(featureLabel)) {
                    return true;
                }
            }
        }
        
        return false;
    }
}

