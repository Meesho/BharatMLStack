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

    public static class DataPath {

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
}


