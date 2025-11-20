package com.bharatml.featurestore.connector;

import com.bharatml.featurestore.core.FeatureEvent;
import com.bharatml.featurestore.core.FeatureConverter;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.api.common.typeutils.TypeSerializer;
import org.apache.flink.streaming.api.functions.sink.TwoPhaseCommitSinkFunction;
import org.apache.kafka.clients.producer.KafkaProducer;
import org.apache.kafka.clients.producer.ProducerRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import persist.QueryOuterClass;

import java.util.Properties;
import java.util.UUID;

/**
 * Transactional Kafka sink using TwoPhaseCommitSinkFunction for exactly-once semantics.
 * 
 * This sink integrates with Flink checkpoints to provide exactly-once delivery guarantees
 * using Kafka transactions.
 * 
 * Requirements:
 * - Flink 1.20.1 or later
 * - Kafka broker with transactions enabled (default in Kafka 0.11+)
 * - Checkpointing enabled in Flink job
 * 
 * Usage:
 * <pre>
 * FeatureStoreClientConfig config = FeatureStoreClientConfig.builder()
 *     .bootstrapServers("localhost:9092")
 *     .topic("my-topic")
 *     .transactional(true)
 *     .build();
 * 
 * DataStream&lt;FeatureEvent&gt; events = ...;
 * env.enableCheckpointing(60000); // Required for exactly-once
 * events.addSink(new FeatureStoreTransactionalKafkaSink(config));
 * </pre>
 * 
 * TODO: In production, generate transactional.id deterministically using:
 * - jobId (from RuntimeContext.getJobId())
 * - subtask index (from RuntimeContext.getIndexOfThisSubtask())
 * - Format: "flink-feature-store-{jobId}-{subtaskIndex}"
 * This ensures idempotency across job restarts and prevents transaction coordinator conflicts.
 * Current implementation uses UUID which may cause issues if the same task restarts.
 */
public class FeatureStoreTransactionalKafkaSink 
        extends TwoPhaseCommitSinkFunction<FeatureEvent, KafkaProducer<String, byte[]>, Void> {
    
    private static final Logger LOG = LoggerFactory.getLogger(FeatureStoreTransactionalKafkaSink.class);
    
    private final FeatureStoreClientConfig config;
    private final String topic;
    private transient String transactionalId;

    @SuppressWarnings("unchecked")
    public FeatureStoreTransactionalKafkaSink(FeatureStoreClientConfig config) {
        super(
            (TypeSerializer<KafkaProducer<String, byte[]>>) (TypeSerializer<?>) 
                TypeInformation.of((Class<KafkaProducer>) (Class<?>) KafkaProducer.class)
                    .createSerializer(new org.apache.flink.api.common.ExecutionConfig()),
            TypeInformation.of(Void.class).createSerializer(new org.apache.flink.api.common.ExecutionConfig())
        );
        this.config = config;
        config.validate();
        if (!config.isTransactional()) {
            throw new IllegalArgumentException("FeatureStoreClientConfig must have transactional=true for FeatureStoreTransactionalKafkaSink");
        }
        this.topic = config.getTopic();
    }

    @Override
    protected KafkaProducer<String, byte[]> beginTransaction() throws Exception {
        // Generate transactional ID
        // TODO: In production, use deterministic ID: jobId + subtaskIndex
        // String jobId = getRuntimeContext().getJobId().toString();
        // int subtaskIndex = getRuntimeContext().getIndexOfThisSubtask();
        // transactionalId = String.format("flink-feature-store-%s-%d", jobId, subtaskIndex);
        transactionalId = "flink-feature-store-" + UUID.randomUUID().toString();
        
        Properties props = config.toProducerProps(true);
        
        // Create transactional producer using factory
        KafkaProducer<String, byte[]> producer = 
            config.getProducerFactory().createProducer(props, true, transactionalId);
        
        // Initialize transactions
        producer.initTransactions();
        
        // Begin transaction
        producer.beginTransaction();
        
        LOG.debug("Began Kafka transaction with ID: {}", transactionalId);
        
        return producer;
    }

    @Override
    protected void invoke(KafkaProducer<String, byte[]> transaction, FeatureEvent value, Context context) throws Exception {
        try {
            // Convert FeatureEvent to Protobuf Query
            QueryOuterClass.Query query = FeatureConverter.toProto(value);
            
            // Serialize to bytes
            byte[] protobufBytes = query.toByteArray();
            
            // Create producer record
            // Use entity_label as key for partitioning
            String key = value.getEntityLabel() != null ? value.getEntityLabel() : "default";
            ProducerRecord<String, byte[]> record = new ProducerRecord<>(topic, key, protobufBytes);
            
            // Send into transaction (not committed yet)
            transaction.send(record);
            
            LOG.debug("Added feature event to transaction: entity={}, topic={}", 
                    value.getEntityLabel(), topic);
        } catch (Exception e) {
            LOG.error("Failed to convert or add feature event to transaction", e);
            throw e;
        }
    }

    @Override
    protected void preCommit(KafkaProducer<String, byte[]> transaction) throws Exception {
        // Flush pending records (optional - Kafka handles this on commit)
        transaction.flush();
        LOG.debug("Pre-commit: flushed transaction {}", transactionalId);
    }

    @Override
    protected void commit(KafkaProducer<String, byte[]> transaction) {
        try {
            transaction.commitTransaction();
            LOG.debug("Committed Kafka transaction: {}", transactionalId);
        } catch (Exception e) {
            LOG.error("Failed to commit Kafka transaction: {}", transactionalId, e);
            throw new RuntimeException("Transaction commit failed", e);
        } finally {
            transaction.close();
        }
    }

    @Override
    protected void abort(KafkaProducer<String, byte[]> transaction) {
        try {
            transaction.abortTransaction();
            LOG.debug("Aborted Kafka transaction: {}", transactionalId);
        } catch (Exception e) {
            LOG.error("Failed to abort Kafka transaction: {}", transactionalId, e);
        } finally {
            transaction.close();
        }
    }
}

