package com.bharatml.featurestore.kafka;

import org.apache.kafka.clients.producer.KafkaProducer;
import org.apache.kafka.clients.producer.ProducerConfig;
import org.apache.kafka.common.serialization.ByteArraySerializer;
import org.apache.kafka.common.serialization.StringSerializer;

import java.util.Properties;

/**
 * Default implementation of KafkaProducerFactory.
 * Sets up standard producer configuration including idempotence and transactional settings.
 */
public class DefaultKafkaProducerFactory implements KafkaProducerFactory {

    @Override
    public KafkaProducer<String, byte[]> createProducer(Properties props, boolean transactional, String transactionalId) {
        Properties producerProps = new Properties();
        producerProps.putAll(props);
        
        // Set serializers
        producerProps.setProperty(ProducerConfig.KEY_SERIALIZER_CLASS_CONFIG, StringSerializer.class.getName());
        producerProps.setProperty(ProducerConfig.VALUE_SERIALIZER_CLASS_CONFIG, ByteArraySerializer.class.getName());
        
        // Set reliability defaults
        producerProps.setProperty(ProducerConfig.ACKS_CONFIG, "all");
        producerProps.setProperty(ProducerConfig.RETRIES_CONFIG, "3");
        producerProps.setProperty(ProducerConfig.ENABLE_IDEMPOTENCE_CONFIG, "true");
        
        // Transactional configuration
        if (transactional) {
            if (transactionalId == null || transactionalId.isEmpty()) {
                throw new IllegalArgumentException("transactionalId is required when transactional=true");
            }
            producerProps.setProperty(ProducerConfig.TRANSACTIONAL_ID_CONFIG, transactionalId);
            // Required for transactions
            producerProps.setProperty(ProducerConfig.MAX_IN_FLIGHT_REQUESTS_PER_CONNECTION, "5");
        }
        
        return new KafkaProducer<>(producerProps);
    }
}

