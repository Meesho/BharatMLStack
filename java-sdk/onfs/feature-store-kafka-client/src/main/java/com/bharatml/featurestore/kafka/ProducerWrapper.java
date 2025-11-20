package com.bharatml.featurestore.kafka;

import org.apache.kafka.clients.producer.Callback;
import org.apache.kafka.clients.producer.KafkaProducer;
import org.apache.kafka.clients.producer.ProducerRecord;
import org.apache.kafka.clients.producer.RecordMetadata;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.Future;

/**
 * Small helper utility for sending Kafka records with callbacks and logging.
 */
public class ProducerWrapper {
    private static final Logger LOG = LoggerFactory.getLogger(ProducerWrapper.class);

    /**
     * Sends a record asynchronously with a callback that logs success/failure.
     * 
     * @param producer KafkaProducer to use
     * @param record ProducerRecord to send
     * @return Future for the send operation
     */
    public static Future<RecordMetadata> sendWithCallback(
            KafkaProducer<String, byte[]> producer, 
            ProducerRecord<String, byte[]> record) {
        return producer.send(record, new Callback() {
            @Override
            public void onCompletion(RecordMetadata metadata, Exception exception) {
                if (exception != null) {
                    LOG.error("Failed to send record to Kafka: topic={}, key={}", 
                            record.topic(), record.key(), exception);
                } else {
                    LOG.debug("Sent record to Kafka: topic={}, partition={}, offset={}, key={}", 
                            metadata.topic(), metadata.partition(), metadata.offset(), record.key());
                }
            }
        });
    }

    /**
     * Sends a record synchronously (blocks until completion).
     * 
     * @param producer KafkaProducer to use
     * @param record ProducerRecord to send
     * @return RecordMetadata for the sent record
     * @throws Exception if send fails
     */
    public static RecordMetadata sendBlocking(
            KafkaProducer<String, byte[]> producer, 
            ProducerRecord<String, byte[]> record) throws Exception {
        return producer.send(record).get();
    }
}

