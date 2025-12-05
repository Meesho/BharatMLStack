package com.bharatml.featurestore.connector;

import java.io.Serializable;
import java.util.Properties;
import java.util.Objects;

public final class FeatureStoreClientConfig implements Serializable {
    private final String bootstrapServers;
    private final String topic;
    private final boolean transactional;
    private final Properties producerProperties;

    private FeatureStoreClientConfig(String bootstrapServers, String topic, boolean transactional, Properties producerProperties) {
        this.bootstrapServers = Objects.requireNonNull(bootstrapServers);
        this.topic = Objects.requireNonNull(topic);
        this.transactional = transactional;
        this.producerProperties = producerProperties != null ? producerProperties : new Properties();
    }

    public static Builder builder() { return new Builder(); }

    public String getBootstrapServers() { return bootstrapServers; }
    public String getTopic() { return topic; }
    public boolean isTransactional() { return transactional; }
    public Properties getProducerProperties() { return producerProperties; }

    public void validate() {
        if (bootstrapServers == null || bootstrapServers.isEmpty()) {
            throw new IllegalArgumentException("bootstrapServers is required");
        }
        if (topic == null || topic.isEmpty()) {
            throw new IllegalArgumentException("topic is required");
        }
    }

    public static class Builder {
        private String bootstrapServers;
        private String topic;
        private boolean transactional = false;
        private Properties props = new Properties();

        public Builder bootstrapServers(String bs) { this.bootstrapServers = bs; return this; }
        public Builder topic(String t) { this.topic = t; return this; }
        public Builder transactional(boolean tx) { this.transactional = tx; return this; }
        public Builder producerProperties(Properties p) { this.props = p; return this; }

        public FeatureStoreClientConfig build() {
            return new FeatureStoreClientConfig(bootstrapServers, topic, transactional, props);
        }
    }
}
