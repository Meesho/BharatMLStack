package com.bharatmlstack.service;

import com.bharatmlstack.BharatMLClient;
import com.bharatmlstack.retrieve.Query;
import com.bharatmlstack.retrieve.FeatureGroup;
import com.bharatmlstack.retrieve.Keys;
import com.bharatmlstack.retrieve.Result;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.HashMap;
import java.util.Map;

@RestController
public class FeatureStoreController {

        private final BharatMLClient client;

    public FeatureStoreController(BharatMLClient client) {
        this.client = client;
    }

    @PostMapping("/retrieve-features")
    public ResponseEntity<Map<String, Object>> retrieveFeatures() {
        Map<String, Object> response = new HashMap<>();
        
        try {

            Query request = Query.newBuilder()
                    .setEntityLabel("catalog")
                    .addFeatureGroups(
                            FeatureGroup.newBuilder()
                                    .setLabel("derived_fp32")
                                    .addFeatureLabels("clicks_by_views_3_days")
                                    .build()
                    )
                    .addKeysSchema("catalog_id")
                    .addKeys(
                            Keys.newBuilder().addCols("176").build()
                    )
                    .addKeys(
                            Keys.newBuilder().addCols("179").build()
                    )
                    .build();

            Result result = client.retrieveFeatures(request);
            
            response.put("success", true);
            response.put("data", result.toString());
            response.put("message", "Features retrieved successfully");
            
            return ResponseEntity.ok(response);
            
        } catch (Exception e) {
            response.put("success", false);
            response.put("error", e.getMessage());
            response.put("message", "Failed to retrieve features");
            return ResponseEntity.status(500).body(response);
        }
    }
}

