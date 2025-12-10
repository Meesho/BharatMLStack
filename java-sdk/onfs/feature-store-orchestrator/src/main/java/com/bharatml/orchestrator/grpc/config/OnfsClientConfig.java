package com.bharatml.orchestrator.grpc.config;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;
import lombok.experimental.SuperBuilder;


@Data
@AllArgsConstructor
@NoArgsConstructor
@Builder
public class OnfsClientConfig {

    private String host;
    private Integer port;
    private Http2Config http2Config;
    private String callerId;
    private String callerToken;

    @Data
    @AllArgsConstructor
    @NoArgsConstructor
    @SuperBuilder
    public static class Http2Config {
        private int connectTimeout;
        private int grpcDeadline;
        private int connectionRequestTimeout;
        private int keepAliveTime;
        private int poolSize;
        private int threadPoolSize;
        private int boundedQueueSize;
        private boolean isPlainText;

        public void setIsPlainText(boolean isPlainText) {
            this.isPlainText = isPlainText;
        }
    }
}
