package com.bharatml.featurestore.connector;

import com.bharatml.featurestore.core.FeatureEvent;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.base.DeliveryGuarantee;
import java.util.Properties;

/**
 * Small helper to build a KafkaSink<FeatureEvent> that writes protobuf bytes.
 * - cfgBootstrapServers: comma separated (e.g. "host:9092")
 * - topic: topic name
 * - producerProps: additional kafka producer config (e.g. acks, retries)
 */
public final class KafkaSinkFactory {

    private KafkaSinkFactory() {}

    /**
     * Create a KafkaSink for FeatureEvent objects.
     *
     * @param cfgBootstrapServers comma-separated bootstrap servers (e.g. "localhost:9092")
     * @param topic               destination topic
     * @param producerProps       optional Kafka producer properties (can be null)
     * @param exactlyOnce         true -> EXACTLY_ONCE, false -> AT_LEAST_ONCE
     * @param transactionalIdPrefix optional transactional id prefix (used when exactlyOnce=true)
     * @return configured KafkaSink<FeatureEvent>
     */
    public static KafkaSink<FeatureEvent> create(
            String cfgBootstrapServers,
            String topic,
            Properties producerProps,
            boolean exactlyOnce,
            String transactionalIdPrefix
    ) {

        // Build record serializer: value (protobuf bytes) and key serializer (entity label)
        KafkaRecordSerializationSchema<FeatureEvent> recordSerializer =
            KafkaRecordSerializationSchema.builder()
                .setTopic(topic)
                .setValueSerializationSchema(new FeatureEventSerialization())
                // pass key serializer implementation (must implement SerializationSchema<T>)
                .setKeySerializationSchema(new FeatureEventKeySerialization())
                .build();


        // use local var to avoid referencing a builder type that may be in a different package
        var builder = KafkaSink.<FeatureEvent>builder()
                .setBootstrapServers(cfgBootstrapServers)
                .setRecordSerializer(recordSerializer)
                .setDeliveryGuarantee(exactlyOnce ? DeliveryGuarantee.EXACTLY_ONCE
                    : DeliveryGuarantee.AT_LEAST_ONCE);

        if (producerProps != null && !producerProps.isEmpty()) {
            builder.setKafkaProducerConfig(producerProps);
        }

        if (exactlyOnce && transactionalIdPrefix != null && !transactionalIdPrefix.isEmpty()) {
            builder.setTransactionalIdPrefix(transactionalIdPrefix);
        }

        return builder.build();
    }
}
