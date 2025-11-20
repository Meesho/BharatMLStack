package com.bharatml.featurestore.connector;

import com.bharatml.featurestore.core.FeatureEvent;
import com.bharatml.featurestore.core.FeatureConverter;
import com.bharatml.featurestore.kafka.ProducerWrapper;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.streaming.api.functions.sink.RichSinkFunction;
import org.apache.kafka.clients.producer.KafkaProducer;
import org.apache.kafka.clients.producer.ProducerRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import persist.QueryOuterClass;

import java.util.Properties;

/**
 * Simple Kafka sink for FeatureEvent using RichSinkFunction.
 * Provides at-least-once semantics (easy to run and debug).
 * 
 * For exactly-once semantics, use FeatureStoreTransactionalSink instead.
 */
public class FeatureStoreKafkaSink extends RichSinkFunction<FeatureEvent> {
    private static final Logger LOG = LoggerFactory.getLogger(FeatureStoreKafkaSink.class);

    private transient KafkaProducer<String, byte[]> producer;
    private final FeatureStoreClientConfig config;
    private final String topic;

    public FeatureStoreKafkaSink(FeatureStoreClientConfig config) {
        this.config = config;
        config.validate();
        this.topic = config.getTopic();
    }

    @Override
    public void open(Configuration parameters) throws Exception {
        super.open(parameters);
        
        Properties props = config.toProducerProps(false);
        
        // Create non-transactional producer using factory
        producer = config.getProducerFactory().createProducer(props, false, null);
        
        LOG.info("Opened Kafka producer for topic: {} (at-least-once semantics)", topic);
    }

    @Override
    public void invoke(FeatureEvent value, Context context) throws Exception {
        try {
            // Convert FeatureEvent to Protobuf Query
            QueryOuterClass.Query query = FeatureConverter.toProto(value);
            
            // Serialize to bytes
            byte[] protobufBytes = query.toByteArray();
            
            // Create producer record
            // Use entity_label as key for partitioning
            String key = value.getEntityLabel() != null ? value.getEntityLabel() : "default";
            ProducerRecord<String, byte[]> record = new ProducerRecord<>(topic, key, protobufBytes);
            
            // Send with callback (async, logs on failure)
            ProducerWrapper.sendWithCallback(producer, record);
        } catch (Exception e) {
            LOG.error("Failed to convert or send feature event to Kafka", e);
            throw e;
        }
    }

    @Override
    public void close() throws Exception {
        if (producer != null) {
            producer.flush();
            producer.close();
            LOG.info("Closed Kafka producer for topic: {}", topic);
        }
        super.close();
    }
}

