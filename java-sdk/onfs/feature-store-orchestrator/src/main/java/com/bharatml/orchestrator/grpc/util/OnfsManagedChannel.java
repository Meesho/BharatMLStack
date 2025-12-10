package com.bharatml.orchestrator.grpc.util;

import com.bharatml.orchestrator.grpc.config.OnfsClientConfig;
import com.bharatml.orchestrator.Constants;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Tag;
import io.micrometer.core.instrument.binder.jvm.ExecutorServiceMetrics;
import io.prometheus.client.CollectorRegistry;
import lombok.Data;
import lombok.extern.slf4j.Slf4j;
import me.dinowernli.grpc.prometheus.Configuration;
import me.dinowernli.grpc.prometheus.MonitoringClientInterceptor;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.stereotype.Component;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.LinkedBlockingDeque;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

@Data
@Slf4j
@Component
@org.springframework.context.annotation.Configuration
public class OnfsManagedChannel {

    @Bean(Constants.ONFS_GRPC_CHANNEL_TEMPLATE)
    @ConditionalOnProperty(prefix = "grpc", name = "onfs-enabled", havingValue = "true")
    public ManagedChannel initializeChannelWithMetrics(@Qualifier(Constants.ONFS_MONITORING_CLIENT_INTERCEPTOR_TEMPLATE) MonitoringClientInterceptor monitoringInterceptor,
                                                       @Qualifier(Constants.ONFS_GRPC_PROPERTIES) OnfsClientConfig grpcClientConfig,
                                                       MeterRegistry meterRegistry) {
        BlockingQueue<Runnable> linkedBlockingDeque = new LinkedBlockingDeque<>(
                grpcClientConfig.getHttp2Config().getBoundedQueueSize());
        ExecutorService executorService = new ThreadPoolExecutor(grpcClientConfig.getHttp2Config().getThreadPoolSize(),
                grpcClientConfig.getHttp2Config().getThreadPoolSize(),
                0L,
                TimeUnit.MILLISECONDS,
                linkedBlockingDeque,
                new ThreadPoolExecutor.AbortPolicy());
        if (grpcClientConfig.getHttp2Config().isPlainText()) {
            return ManagedChannelBuilder.forAddress(grpcClientConfig.getHost(), grpcClientConfig.getPort())
                    .executor(ExecutorServiceMetrics.monitor(
                            meterRegistry,
                            executorService,
                            "onfs::channel_thread",
                            Tag.of("instance", "onfs::channel_thread")
                    ))
                    .intercept(monitoringInterceptor)
                    .defaultLoadBalancingPolicy("round_robin")
                    .usePlaintext()
                    .idleTimeout(Long.MAX_VALUE, TimeUnit.NANOSECONDS)
                    .keepAliveWithoutCalls(true)
                    .build();
        } else {
            return ManagedChannelBuilder.forAddress(grpcClientConfig.getHost(), grpcClientConfig.getPort())
                    .executor(ExecutorServiceMetrics.monitor(
                            meterRegistry,
                            executorService,
                            "onfs::channel_thread",
                            Tag.of("instance", "onfs::channel_thread")
                    ))
                    .intercept(monitoringInterceptor)
                    .idleTimeout(Long.MAX_VALUE, TimeUnit.NANOSECONDS)
                    .keepAliveWithoutCalls(true)
                    .build();
        }
    }

    @Bean(Constants.ONFS_MONITORING_CLIENT_INTERCEPTOR_TEMPLATE)
    @ConditionalOnProperty(prefix = "grpc", name = "onfs-enabled", havingValue = "true")
    @ConditionalOnMissingBean
    public static MonitoringClientInterceptor intializeMonitoringClientInterceptor(CollectorRegistry collectorRegistry) {
        return
                MonitoringClientInterceptor.create(Configuration.allMetrics()
                        .withCollectorRegistry(collectorRegistry));
    }

    @Bean(Constants.ONFS_GRPC_PROPERTIES)
    @ConditionalOnProperty(prefix = "grpc", name = "onfs-enabled", havingValue = "true")
    @ConditionalOnMissingBean
    @ConfigurationProperties(prefix = "client.onfs-grpc")
    public OnfsClientConfig getFeatureStoreClientConfig() {
        return new OnfsClientConfig();
    }

}
