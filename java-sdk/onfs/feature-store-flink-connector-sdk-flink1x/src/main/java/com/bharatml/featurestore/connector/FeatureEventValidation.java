package com.bharatml.featurestore.connector;

import com.bharatml.featurestore.connector.horizon.SourceMappingResponse;
import com.bharatml.featurestore.connector.horizon.SourceMappingHolder;
import com.bharatml.featurestore.core.FeatureEvent;
import org.apache.flink.api.common.functions.RichFilterFunction;

import java.util.*;

public final class FeatureEventValidation extends RichFilterFunction<FeatureEvent> {
    
    public FeatureEventValidation() {
        // No-arg constructor - reads from SourceMappingHolder.get() in filter() method
    }

    @Override
    public boolean filter(FeatureEvent event){
        // Read latest schema from shared holder (updated by etcd watcher)
        SourceMappingResponse sourceMapping = SourceMappingHolder.get();
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
