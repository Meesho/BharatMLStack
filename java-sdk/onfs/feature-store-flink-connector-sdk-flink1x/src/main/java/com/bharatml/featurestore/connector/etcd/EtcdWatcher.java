package com.bharatml.featurestore.connector.etcd;

import com.bharatml.featurestore.connector.HorizonClient;
import com.bharatml.featurestore.connector.horizon.SourceMappingHolder;
import com.bharatml.featurestore.connector.horizon.SourceMappingResponse;
import io.etcd.jetcd.ByteSequence;
import io.etcd.jetcd.Client;
import io.etcd.jetcd.options.WatchOption;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.util.concurrent.atomic.AtomicReference;

/**
 * EtcdWatcher monitors an etcd key for changes and updates feature metadata
 * by calling Horizon API when changes are detected.
 * 
 * Usage:
 * <pre>
 * AtomicReference&lt;SourceMappingResponse&gt; metadataRef = new AtomicReference&lt;&gt;();
 * EtcdWatcher watcher = new EtcdWatcher(
 *     "http://localhost:2379",
 *     "username",
 *     "password",
 *     "/feature/schema",
 *     "http://horizon:8080",
 *     "jobId",
 *     "jobToken",
 *     metadataRef
 * );
 * watcher.start();
 * // ... later ...
 * watcher.stop();
 * </pre>
 */
public class EtcdWatcher {
    private static final Logger logger = LoggerFactory.getLogger(EtcdWatcher.class);
    
    private final String etcdEndpoint;
    private final String etcdUsername;
    private final String etcdPassword;
    private final String watchKey;
    private final String horizonBaseUrl;
    private final String jobId;
    private final String jobToken;
    
    private Client etcdClient;
    private volatile boolean isWatching = false;
    private Thread watchThread;
    
    /**
     * Creates a new EtcdWatcher instance.
     * 
     * @param etcdEndpoint etcd server endpoint
     * @param etcdUsername etcd username for authentication
     * @param etcdPassword etcd password for authentication
     * @param watchKey etcd key to watch for changes
     * @param horizonBaseUrl Horizon API base URL
     * @param jobId job ID for Horizon API calls
     * @param jobToken job token for Horizon API calls
     */
    public EtcdWatcher(
            String etcdEndpoint,
            String etcdUsername,
            String etcdPassword,
            String watchKey,
            String horizonBaseUrl,
            String jobId,
            String jobToken) {
        this.etcdEndpoint = etcdEndpoint;
        this.etcdUsername = etcdUsername;
        this.etcdPassword = etcdPassword;
        this.watchKey = watchKey;
        this.horizonBaseUrl = horizonBaseUrl;
        this.jobId = jobId;
        this.jobToken = jobToken;
    }
    
    /**
     * Starts watching the etcd key for changes.
     * This method is non-blocking and starts a background thread.
     * 
     * @throws IllegalStateException if already watching
     * @throws RuntimeException if etcd client initialization fails
     */
    public void start() {
        if (isWatching) {
            throw new IllegalStateException("EtcdWatcher is already watching");
        }
        
        try {
            // Initialize etcd client
            etcdClient = Client.builder()
                    .endpoints(etcdEndpoint)
                    .user(ByteSequence.from(etcdUsername.getBytes()))
                    .password(ByteSequence.from(etcdPassword.getBytes()))
                    .build();
            
            // Start watching in a background thread
            isWatching = true;
            watchThread = new Thread(this::watchLoop, "EtcdWatcher-" + watchKey);
            watchThread.setDaemon(true);
            watchThread.start();
            
            logger.info("Started watching etcd key: {}", watchKey);
        } catch (Exception e) {
            isWatching = false;
            logger.error("Failed to start etcd watcher", e);
            throw new RuntimeException("Failed to start etcd watcher: " + e.getMessage(), e);
        }
    }
    
    /**
     * Stops watching the etcd key and closes the etcd client.
     */
    public void stop() {
        if (!isWatching) {
            return;
        }
        
        isWatching = false;
        
        if (etcdClient != null) {
            try {
                etcdClient.close();
                logger.info("Stopped watching etcd key: {}", watchKey);
            } catch (Exception e) {
                logger.warn("Error closing etcd client", e);
            }
        }
        
        if (watchThread != null) {
            watchThread.interrupt();
        }
    }
    
    /**
     * Main watch loop that monitors etcd for changes.
     */
    private void watchLoop() {
        try {
            ByteSequence key = ByteSequence.from(watchKey.getBytes());
            
            // Watch for PUT + DELETE events on this key
            WatchOption options = WatchOption.builder()
                    .isPrefix(false)   // Watch specific key, not prefix
                    .build();
            
            etcdClient.getWatchClient().watch(key, options, response -> {
                if (!response.getEvents().isEmpty()) {
                    logger.info("🔄 Change detected in etcd key: {}", watchKey);
                    updateFeatureMetadata();
                }
            });
            
            logger.info("👀 Watching etcd key: {}", watchKey);
            
            // Keep the watcher running until stopped
            while (isWatching) {
                Thread.sleep(1000);
            }
        } catch (InterruptedException e) {
            logger.info("EtcdWatcher thread interrupted, stopping watch");
            Thread.currentThread().interrupt();
        } catch (Exception e) {
            logger.error("Error in etcd watch loop", e);
            isWatching = false;
        }
    }
    
    /**
     * Updates the feature metadata by calling Horizon API.
     * This method is thread-safe and updates the AtomicReference.
     */
    private void updateFeatureMetadata() {
        try {
            logger.info("Fetching updated feature metadata from Horizon (jobId: {})", jobId);
            
            HorizonClient horizonClient = new HorizonClient(horizonBaseUrl);
            SourceMappingResponse updatedResponse = horizonClient.getHorizonResponse(jobId, jobToken);

            //Automatically updates with the bew response
            SourceMappingHolder.update(updatedResponse);

            logger.info("Successfully updated feature metadata from Horizon");
        } catch (IOException | InterruptedException e) {
            logger.error("Failed to update feature metadata from Horizon: {}", e.getMessage(), e);
            // Don't throw exception - allow watcher to continue
        }
    }
    
    /**
     * Checks if the watcher is currently active.
     * 
     * @return true if watching, false otherwise
     */
    public boolean isWatching() {
        return isWatching;
    }
}
