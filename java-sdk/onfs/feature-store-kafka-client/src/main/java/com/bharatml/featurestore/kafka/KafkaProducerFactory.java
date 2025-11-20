package com.bharatml.featurestore.kafka;

import org.apache.kafka.clients.producer.KafkaProducer;

import java.util.Properties;

/**
 * Factory interface for creating KafkaProducer instances.
 * Allows injection of custom producer creation logic for testing or specialized configurations.
 */
public interface KafkaProducerFactory {
    /**
     * Creates a KafkaProducer with the given properties.
     * 
     * @param props Kafka producer properties (bootstrap.servers, etc.)
     * @param transactional Whether to enable transactions
     * @param transactionalId Transactional ID (required if transactional=true)
     * @return Configured KafkaProducer
     */
    KafkaProducer<String, byte[]> createProducer(Properties props, boolean transactional, String transactionalId);
}

