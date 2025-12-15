package com.bharatml.featurestore.connector.horizon;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Represents a single feature mapping with its metadata.
 */
public class DataPath {
    
    @JsonProperty("entity-label")
    private String entityLabel;
    
    @JsonProperty("feature-group-label")
    private String featureGroupLabel;
    
    @JsonProperty("feature-label")
    private String featureLabel;
    
    @JsonProperty("source-data-column")
    private String sourceDataColumn;
    
    @JsonProperty("default-value")
    private String defaultValue;
    
    @JsonProperty("data-type")
    private String dataType;
    
    public DataPath() {
    }
    
    public String getEntityLabel() {
        return entityLabel;
    }
    
    public String getFeatureGroupLabel() {
        return featureGroupLabel;
    }
    
    public String getFeatureLabel() {
        return featureLabel;
    }
    
    public String getSourceDataColumn() {
        return sourceDataColumn;
    }
    
    public String getDefaultValue() {
        return defaultValue;
    }
    
    public String getDataType() {
        return dataType;
    }
}

