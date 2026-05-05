package com.bharatml.featurestore.connector;

import com.bharatml.featurestore.connector.horizon.SourceMappingResponse;
import com.bharatml.featurestore.connector.horizon.SourceMappingHolder;
import com.bharatml.featurestore.core.FeatureEvent;
import org.apache.flink.api.common.functions.RichFilterFunction;
import org.apache.flink.configuration.Configuration;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

public final class FeatureEventValidation extends RichFilterFunction<FeatureEvent> {

    private static final Logger log = LoggerFactory.getLogger(FeatureEventValidation.class);
    private transient SourceMappingHolder sourceMappingHolder;

    private static final Map<String, SourceMappingHolder> holderRegistry = new java.util.concurrent.ConcurrentHashMap<>();
    private final String holderKey;

    private transient long validEventsCount;
    private transient long invalidEventsCount;
    
    /**
     * Creates a FeatureEventValidation filter that uses a SourceMappingHolder
     * identified by the given key.
     *
     * @param holderKey unique key to identify the SourceMappingHolder in the registry
     */
    public FeatureEventValidation(String holderKey) {
        this.holderKey = holderKey;
    }
    
    /**
     * Registers a SourceMappingHolder with a unique key.
     * This should be called before creating the Flink job.
     *
     * @param key unique identifier for the holder
     * @param holder the SourceMappingHolder instance
     */
    public static void registerHolder(String key, SourceMappingHolder holder) {
        holderRegistry.put(key, holder);
    }
    
    /**
     * Unregisters a SourceMappingHolder.
     *
     * @param key unique identifier for the holder
     */
    public static void unregisterHolder(String key) {
        holderRegistry.remove(key);
    }
    
    @Override
    public void open(Configuration parameters) throws Exception {
        super.open(parameters);

        this.sourceMappingHolder = holderRegistry.get(holderKey);
        if (sourceMappingHolder == null) {
            throw new IllegalStateException("SourceMappingHolder not found for key: " + holderKey);
        }

        this.validEventsCount = 0;
        this.invalidEventsCount = 0;
    }

    @Override
    public boolean filter(FeatureEvent event){
        // Get source mapping from holder
        SourceMappingResponse sourceMapping = sourceMappingHolder.get();
        if (sourceMapping == null) {
            return false;
        }

        // Get keysSchema map from response (Map<entityLabel, List<keySchema>>)
        Map<String, List<String>> responseKeysSchema = sourceMapping.getKeys();
        
        // Validate keys schema - check if event's keysSchema list is present in response's keysSchema map
        if (!isValidKeysSchema(event, responseKeysSchema)) {
            invalidEventsCount++;
            log.error("Validation failed due to invalid event keys-schema");
            return false;
        }

        // Check if entity label, feature group, and feature labels exist in the data
        if (sourceMapping.hasFeatureMapping(
                event.getEntityLabel(),
                event.getFeatureGroupLabel(),
                event.getFeatureLabels())
        ) {
            return true;
        }
        invalidEventsCount++;
        log.error("Validation failed due to either invalid entityLabel, featureGroupLabel or featureLabel");
        return false;
    }
    
    public boolean isValidKeysSchema(FeatureEvent event, Map<String, List<String>> responseKeysSchema){
        // Check if response keys map is null or empty
        if (responseKeysSchema == null || responseKeysSchema.isEmpty()) {
            return false;
        }

        // Check if event's keysSchema is null or empty
        List<String> eventKeysSchema = event.getKeysSchema();
        if (eventKeysSchema == null || eventKeysSchema.isEmpty()) {
            return false;
        }

        // Check if event's entityLabel exists in the response keys map
        String entityLabel = event.getEntityLabel();
        if (entityLabel == null) {
            return false;
        }

        // Get the list of key schemas for this entity label from the response
        List<String> responseKeysSchemaForEventLabel = responseKeysSchema.get(entityLabel);
        if (responseKeysSchemaForEventLabel == null || responseKeysSchemaForEventLabel.isEmpty()) {
            return false;
        }

        // Check if all keys in event's keysSchema are present in the response's keysSchema list
        HashSet<String> keySchemaSet = new HashSet<>(responseKeysSchemaForEventLabel);
        return keySchemaSet.containsAll(eventKeysSchema);
    }

    public long getValidEventsCount() {
        return validEventsCount;
    }

    public long getInvalidEventsCount() {
        return invalidEventsCount;
    }
}
