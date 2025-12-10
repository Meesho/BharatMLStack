Online Feature Store Java Client

this is grpc library for connecting to Online Feature Store Service
to easily utilize this library into any service make sure these things are taken care:

1) add below configs into respective envs

```
grpc:
  onfs-enabled: 'true'
```

```
client:
  onfs-grpc:
    host: online-feature-store-api-secondary.stg.meesho.int
    port: '8080'
    http2-config:
        bounded-queue-size: '100'
        connect-timeout: '100'
        connection-request-timeout: '100'
        grpc-deadline: '200'
        is-plain-text: 'true'
        keep-alive-time: '5000'
        pool-size: '100'
        thread-pool-size: '100'
    caller-id: ${ONLINE_FEATURE_STORE_CALLER_ID:dummy}
    caller-token: ${ONLINE_FEATURE_STORE_AUTH_TOKEN:dummy}
```

2) this is one example how we can impliment a wrapper for grpc call using this java client:


```
package com.example.service.onfs;

import com.bharatml.orchestrator.dtos.DecodedResult;
import com.bharatml.orchestrator.dtos.FeatureGroup;
import com.bharatml.orchestrator.dtos.Keys;
import com.bharatml.orchestrator.dtos.Query;
import com.bharatml.orchestrator.services.impls.OnfsService;
import com.bharatml.orchestrator.grpc.config.OnfsClientConfig;
import static com.bharatml.orchestrator.Constants.ONFS_GRPC_PROPERTIES;
import static com.bharatml.orchestrator.Constants.ONFS_AUTH_TOKEN;
import static com.bharatml.orchestrator.Constants.ONFS_CALLER_ID;
import io.grpc.Metadata;

import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.stereotype.Component;
import org.springframework.stereotype.Service;
import java.util.Arrays;
import java.util.List;

@Component
@Service
@Slf4j
public class OnfsServiceWrapper {
    private final OnfsService onfsService;
    private final OnfsClientConfig onfsClientConfig;

    @Autowired
    public OnfsServiceWrapper(OnfsService onfsService, @Qualifier(ONFS_GRPC_PROPERTIES) OnfsClientConfig onfsClientConfig) {
        this.onfsService = onfsService;
        this.onfsClientConfig = onfsClientConfig;
    }

    // expose a public method to everyone to make it accessible
    public DecodedResult getDecodedFeatureAttributes(List<String> keys, String entityLabel){
        try {
            Query query = buildQuery(keys, entityLabel);
            return retrieveDecodedFeatureResult(query);
        } catch (OnfsGrpcFailureException e){
            log.error("Online Feature Store grpc connection Exception: {}", e.getMessage());
            throw e;
        } catch (Exception e) {
            log.error("Failed to get Decoded attributes for entity: {}, keys: {},  Exception: {}",entityLabel, keys, e.getMessage() );
            throw e;
        }
    }

    private DecodedResult retrieveDecodedFeatureResult(Query query) {             
        try {
            Metadata metadata = buildMetadata();
            DecodedResult decodedResult = onfsService.retrieveDecodedResult(metadata, query);
            return decodedResult;
        } catch (Exception e) {
            throw new OnfsGrpcFailureException("Failed to retrieve decoded result from Online Feature Store, Exception: " + e.getMessage());
        }
    }

    // build metadata
    private Metadata buildMetadata() {
        Metadata metadata = new Metadata();
        metadata.put(Metadata.Key.of(ONFS_CALLER_ID, Metadata.ASCII_STRING_MARSHALLER),
            onfsClientConfig.getCallerId());
        metadata.put(Metadata.Key.of(ONFS_AUTH_TOKEN, Metadata.ASCII_STRING_MARSHALLER),
            onfsClientConfig.getCallerToken());
        return metadata;
    }


    // for building query visit http://trufflebox.prd.meesho.int/  and based on specific needs select label, 
    // for example below it's related to sub_order entityLabel so have selected it's related
    // label to build query
    

    private Query buildQuery(List<String> keys, String entityLabel) {
        // Example query building logic
        // Replace with your actual entity labels and feature groups
        
        FeatureGroup featureGroup1 = FeatureGroup.builder()
                .label("derived_fp32")
                .featureLabels(List.of("feature1", "feature2"))
                .build();

        FeatureGroup featureGroup2 = FeatureGroup.builder()
                .label("derived_bool")
                .featureLabels(List.of("feature3"))
                .build();

        List<String> keySchema = List.of("key_column");

        Keys keyList = Keys.builder()
                .cols(keys)
                .build();

        return Query.builder()
                .entityLabel(entityLabel)
                .featureGroups(Arrays.asList(featureGroup1, featureGroup2))
                .keysSchema(keySchema)
                .keys(List.of(keyList))
                .build();
    }
}
```
