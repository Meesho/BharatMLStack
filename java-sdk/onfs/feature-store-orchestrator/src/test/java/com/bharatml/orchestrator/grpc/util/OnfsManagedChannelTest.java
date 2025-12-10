package com.bharatml.orchestrator.grpc.util;

import com.bharatml.orchestrator.grpc.config.OnfsClientConfig;
import com.bharatml.orchestrator.grpc.config.OnfsClientConfig.Http2Config;
import io.grpc.ManagedChannel;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.MeterRegistry.More;
import me.dinowernli.grpc.prometheus.MonitoringClientInterceptor;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class OnfsManagedChannelTest {

    private OnfsManagedChannel channelConfig;

    @Mock
    private MonitoringClientInterceptor monitoringInterceptor;

    @Mock
    private OnfsClientConfig grpcClientConfig;

    @Mock
    private Http2Config http2Config;

    @Mock
    private MeterRegistry meterRegistry;

    @Mock
    private More meterRegistryMore;

    @BeforeEach
    void setUp() {

    }

    @Test
    void testInitializeChannelWithMetrics_plainTextTrue() {
        channelConfig = new OnfsManagedChannel();
        when(grpcClientConfig.getHttp2Config()).thenReturn(http2Config);
        when(grpcClientConfig.getHost()).thenReturn("localhost");
        when(grpcClientConfig.getPort()).thenReturn(9090);
        when(http2Config.getBoundedQueueSize()).thenReturn(100);
        when(http2Config.getThreadPoolSize()).thenReturn(10);
        when(http2Config.isPlainText()).thenReturn(true);
        when(meterRegistry.more()).thenReturn(meterRegistryMore);


        when(http2Config.isPlainText()).thenReturn(true);
        ManagedChannel channel = channelConfig.initializeChannelWithMetrics(
                monitoringInterceptor, grpcClientConfig, meterRegistry);
        assertNotNull(channel);
        assertFalse(channel.isShutdown());
        channel.shutdownNow();
    }

    @Test
    void testInitializeChannelWithMetrics_plainTextFalse() {
        channelConfig = new OnfsManagedChannel();
        when(grpcClientConfig.getHttp2Config()).thenReturn(http2Config);
        when(grpcClientConfig.getHost()).thenReturn("localhost");
        when(grpcClientConfig.getPort()).thenReturn(9090);
        when(http2Config.getBoundedQueueSize()).thenReturn(100);
        when(http2Config.getThreadPoolSize()).thenReturn(10);
        when(http2Config.isPlainText()).thenReturn(true);
        when(meterRegistry.more()).thenReturn(meterRegistryMore);
        when(http2Config.isPlainText()).thenReturn(false);


        ManagedChannel channel = channelConfig.initializeChannelWithMetrics(
                monitoringInterceptor, grpcClientConfig, meterRegistry);

        assertNotNull(channel);
        assertFalse(channel.isShutdown());
        channel.shutdownNow();
    }

    @Test
    void testIntializeMonitoringClientInterceptor() {
        var collectorRegistry = new io.prometheus.client.CollectorRegistry();
        MonitoringClientInterceptor interceptor = OnfsManagedChannel.intializeMonitoringClientInterceptor(collectorRegistry);
        assertNotNull(interceptor);
    }

    @Test
    void testGetFeatureStoreClientConfig() {
        channelConfig = new OnfsManagedChannel();
        OnfsClientConfig config = channelConfig.getFeatureStoreClientConfig();
        assertNotNull(config);
    }
}



