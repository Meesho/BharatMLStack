package com.bharatml.featurestore.connector;

import com.bharatml.featurestore.core.FeatureEvent;
import org.apache.flink.streaming.api.functions.sink.RichSinkFunction;
import org.apache.flink.streaming.api.functions.sink.TwoPhaseCommitSinkFunction;

/**
 * Factory to create Feature Store sinks from FeatureStoreClientConfig.
 */
public class FeatureSinkFactory {

    /**
     * Creates a simple sink (at-least-once semantics) using RichSinkFunction.
     * 
     * @param cfg FeatureStoreClientConfig with bootstrap servers and topic
     * @return RichSinkFunction for FeatureEvent
     */
    public static RichSinkFunction<FeatureEvent> createSink(FeatureStoreClientConfig cfg) {
        cfg.validate();
        return new FeatureStoreKafkaSink(cfg);
    }

    /**
     * Creates a transactional sink (exactly-once semantics) using TwoPhaseCommitSinkFunction.
     * 
     * @param cfg FeatureStoreClientConfig with bootstrap servers, topic, and transactional=true
     * @return TwoPhaseCommitSinkFunction for FeatureEvent
     */
    public static TwoPhaseCommitSinkFunction<FeatureEvent, ?, ?> createTransactionalSink(FeatureStoreClientConfig cfg) {
        cfg.validate();
        if (!cfg.isTransactional()) {
            throw new IllegalArgumentException("FeatureStoreClientConfig must have transactional=true for transactional sink");
        }
        return new FeatureStoreTransactionalKafkaSink(cfg);
    }
}

