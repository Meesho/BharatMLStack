package com.bharatml.featurestore.connector;

import com.bharatml.featurestore.kafka.DefaultKafkaProducerFactory;
import com.bharatml.featurestore.kafka.KafkaProducerFactory;

import java.util.HashMap;
import java.util.Map;
import java.util.Properties;

/**
 * Configuration builder for Feature Store client.
 * Provides builder pattern for bootstrap servers, topic, acks, retries, transactional flag, 
 * additional properties, and optional KafkaProducerFactory injection.
 */
public class FeatureStoreClientConfig {
    private String bootstrapServers;
    private String topic;
    private String acks = "all";
    private int retries = 3;
    private boolean transactional = false;
    private Map<String, String> additionalProperties = new HashMap<>();
    private KafkaProducerFactory producerFactory;

    private FeatureStoreClientConfig() {
        this.producerFactory = new DefaultKafkaProducerFactory();
    }

    public static FeatureStoreClientConfig builder() {
        return new FeatureStoreClientConfig();
    }

    public FeatureStoreClientConfig bootstrapServers(String bootstrapServers) {
        this.bootstrapServers = bootstrapServers;
        return this;
    }

    public FeatureStoreClientConfig topic(String topic) {
        this.topic = topic;
        return this;
    }

    public FeatureStoreClientConfig acks(String acks) {
        this.acks = acks;
        return this;
    }

    public FeatureStoreClientConfig retries(int retries) {
        this.retries = retries;
        return this;
    }

    public FeatureStoreClientConfig transactional(boolean transactional) {
        this.transactional = transactional;
        return this;
    }

    public FeatureStoreClientConfig additionalProperty(String key, String value) {
        this.additionalProperties.put(key, value);
        return this;
    }

    public FeatureStoreClientConfig additionalProperties(Map<String, String> properties) {
        this.additionalProperties.putAll(properties);
        return this;
    }

    /**
     * Sets a custom KafkaProducerFactory. If not set, DefaultKafkaProducerFactory is used.
     */
    public FeatureStoreClientConfig producerFactory(KafkaProducerFactory factory) {
        this.producerFactory = factory != null ? factory : new DefaultKafkaProducerFactory();
        return this;
    }

    public String getBootstrapServers() {
        return bootstrapServers;
    }

    public String getTopic() {
        return topic;
    }

    public String getAcks() {
        return acks;
    }

    public int getRetries() {
        return retries;
    }

    public boolean isTransactional() {
        return transactional;
    }

    public Map<String, String> getAdditionalProperties() {
        return new HashMap<>(additionalProperties);
    }

    public KafkaProducerFactory getProducerFactory() {
        return producerFactory;
    }

    /**
     * Produces Properties for Kafka producer configuration.
     * 
     * @param transactional Whether the producer will be transactional
     * @return Properties configured for Kafka producer
     */
    public Properties toProducerProps(boolean transactional) {
        Properties props = new Properties();
        props.setProperty("bootstrap.servers", bootstrapServers);
        props.setProperty("acks", acks);
        props.setProperty("retries", String.valueOf(retries));
        
        // Add additional properties
        for (Map.Entry<String, String> entry : additionalProperties.entrySet()) {
            props.setProperty(entry.getKey(), entry.getValue());
        }
        
        return props;
    }

    public FeatureStoreClientConfig build() {
        return this;
    }

    public void validate() {
        if (bootstrapServers == null || bootstrapServers.isEmpty()) {
            throw new IllegalArgumentException("bootstrapServers is required");
        }
        if (topic == null || topic.isEmpty()) {
            throw new IllegalArgumentException("topic is required");
        }
    }
}

