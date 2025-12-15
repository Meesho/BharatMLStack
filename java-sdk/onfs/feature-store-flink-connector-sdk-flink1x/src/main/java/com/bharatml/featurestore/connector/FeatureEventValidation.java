package com.bharatml.featurestore.connector;

import com.bharatml.featurestore.connector.horizon.SourceMappingResponse;
import com.bharatml.featurestore.connector.horizon.SourceMappingHolder;
import com.bharatml.featurestore.core.FeatureEvent;
import org.apache.flink.api.common.functions.RichFilterFunction;
import org.apache.flink.configuration.Configuration;

import java.util.*;

public final class FeatureEventValidation extends RichFilterFunction<FeatureEvent> {
    
    // Make this transient so Flink doesn't serialize it
    private transient SourceMappingHolder sourceMappingHolder;
    
    // Store the holder reference in a static registry (keyed by job name or unique ID)
    // This allows us to pass the holder without serializing it
    private static final Map<String, SourceMappingHolder> holderRegistry = new java.util.concurrent.ConcurrentHashMap<>();
    private final String holderKey;
    
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
        // Retrieve the holder from the registry in open() (runs on task manager)
        this.sourceMappingHolder = holderRegistry.get(holderKey);
        if (sourceMappingHolder == null) {
            throw new IllegalStateException("SourceMappingHolder not found for key: " + holderKey);
        }
    }

    @Override
    public boolean filter(FeatureEvent event){
        SourceMappingResponse sourceMapping = sourceMappingHolder.get();
        if (sourceMapping == null) {
            return false;
        }
        
        Map<String, List<String>> keys = sourceMapping.getKeys();
        if(keys == null) {
            return false;
        }
        List<String> entityKeys = keys.get(event.getEntityLabel());
        if(isValidKeys(event, entityKeys)){
            return sourceMapping.hasFeatureMapping(event.getEntityLabel(), event.getFeatureGroupLabel(), event.getFeatureLabels());
        }
        return false;
    }
    
    public boolean isValidKeys(FeatureEvent event, List<String> keys){
        if(keys == null || keys.isEmpty()) {
            return false;
        }
        Set<String> keySet = new HashSet<>(keys);
        for (String key: event.getKeysSchema()){
            if(!keySet.contains(key)){
                return false;
            }
        }
        return true;
    }  
}
