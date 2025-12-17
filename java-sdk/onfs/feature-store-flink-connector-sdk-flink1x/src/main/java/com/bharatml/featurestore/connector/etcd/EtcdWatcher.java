package com.bharatml.featurestore.connector.etcd;

import com.bharatml.featurestore.connector.horizon.HorizonClient;
import com.bharatml.featurestore.connector.horizon.SourceMappingHolder;
import com.bharatml.featurestore.connector.horizon.SourceMappingResponse;
import io.etcd.jetcd.ByteSequence;
import io.etcd.jetcd.Client;
import io.etcd.jetcd.Watch;
import io.etcd.jetcd.options.WatchOption;
import io.etcd.jetcd.watch.WatchResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.nio.charset.StandardCharsets;

public class EtcdWatcher {

    private static final Logger logger = LoggerFactory.getLogger(EtcdWatcher.class);

    private final String etcdEndpoint;
    private final String etcdUsername;
    private final String etcdPassword;
    private final String watchKey;
    private final HorizonClient horizonClient;
    private final String jobId;
    private final String jobToken;
    private final SourceMappingHolder sourceMappingHolder;

    private Client etcdClient;
    private Watch.Watcher watcher;
    private volatile boolean running;

    public EtcdWatcher(
            String etcdEndpoint,
            String etcdUsername,
            String etcdPassword,
            String watchKey,
            HorizonClient horizonClient,
            String jobId,
            String jobToken,
            SourceMappingHolder sourceMappingHolder
    ) {
        this.etcdEndpoint = etcdEndpoint;
        this.etcdUsername = etcdUsername;
        this.etcdPassword = etcdPassword;
        this.watchKey = watchKey;
        this.horizonClient = horizonClient;
        this.jobId = jobId;
        this.jobToken = jobToken;
        this.sourceMappingHolder = sourceMappingHolder;
    }

    public static Builder builder(){ return new Builder(); }

    public synchronized void start() {
        if (running) {
            throw new IllegalStateException("EtcdWatcher already running");
        }

        try {
            etcdClient = Client.builder()
                    .endpoints(etcdEndpoint)
                    .user(ByteSequence.from(etcdUsername, StandardCharsets.UTF_8))
                    .password(ByteSequence.from(etcdPassword, StandardCharsets.UTF_8))
                    .build();

            WatchOption option = WatchOption.builder()
                    .isPrefix(true)
                    .build();

            ByteSequence key = ByteSequence.from(watchKey, StandardCharsets.UTF_8);

            watcher = etcdClient.getWatchClient().watch(key, option, new Watch.Listener() {

                @Override
                public void onNext(WatchResponse response) {
                    logger.info(
                            "etcd change detected under prefix {}",
                            watchKey
                    );

                    // ANY change triggers mapping refresh by calling Horizon
                    refreshSourceMapping();
                }

                @Override
                public void onError(Throwable throwable) {
                    logger.error("etcd watch error for prefix {}", watchKey, throwable);
                }

                @Override
                public void onCompleted() {
                    logger.info("etcd watch completed for prefix {}", watchKey);
                }
            });

            running = true;
            logger.info("Started etcd watcher for prefix {}", watchKey);

        } catch (Exception e) {
            logger.error("Failed to start etcd watcher for prefix {}", watchKey, e);
            throw new RuntimeException(e);
        }
    }

    public synchronized void stop() {
        running = false;

        if (watcher != null) {
            try {
                watcher.close();
            } catch (Exception e) {
                logger.error("Error closing etcd watcher", e);
            }
        }

        if (etcdClient != null) {
            try {
                etcdClient.close();
            } catch (Exception e) {
                logger.error("Error closing etcd client", e);
            }
        }

        logger.info("Stopped etcd watcher for prefix {}", watchKey);
    }

    private void refreshSourceMapping() {
        try {
            logger.info("Calling Horizon to refresh metadata (jobId={})", jobId);

            SourceMappingResponse response =
                    horizonClient.getHorizonResponse(jobId, jobToken);

            sourceMappingHolder.update(response);

            logger.info("Horizon response updated successfully");

        } catch (IOException | InterruptedException e) {
            logger.error("Failed to refresh metadata from Horizon", e);
        }
    }

    public boolean isRunning() {
        return running;
    }

    public static class Builder {
        private String etcdEndpoint;
        private String etcdUsername;
        private String etcdPassword;
        private String watchKey;
        private HorizonClient horizonClient;
        private String jobId;
        private String jobToken;
        private SourceMappingHolder sourceMappingHolder;

        public Builder etcdEndpoint(String ep) { this.etcdEndpoint = ep; return this; }
        public Builder etcdUsername(String u) { this.etcdUsername = u; return this; }
        public Builder etcdPassword(String p) { this.etcdPassword = p; return this; }
        public Builder watchKey(String w) { this.watchKey = w; return this; }
        public Builder horizonClient(HorizonClient h) { this.horizonClient = h; return this; }
        public Builder jobId(String id) { this.jobId = id; return this; }
        public Builder jobToken(String t) { this.jobToken = t; return this; }
        public Builder sourceMappingHolder(SourceMappingHolder sm) { this.sourceMappingHolder= sm; return this; }

        public EtcdWatcher build() {
            return new EtcdWatcher(etcdEndpoint, etcdUsername, etcdPassword, watchKey, horizonClient, jobId, jobToken, sourceMappingHolder);
        }
    }
}
