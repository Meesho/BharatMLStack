package com.bharatmlstack.service;

import com.bharatmlstack.BharatMLClient;
import com.bharatmlstack.retrieve.Query;
import com.bharatmlstack.retrieve.FeatureGroup;
import com.bharatmlstack.retrieve.Keys;
import com.bharatmlstack.retrieve.Result;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.TimeUnit;

public class Main {
    public static void main(String[] args) {
        System.out.println("Hello, feature-store-java!");

        Map<String, String> metadata = new HashMap<>();
        metadata.put("online-feature-store-auth-token", "test");
        metadata.put("online-feature-store-caller-id", "model-proxy-service-experiment");

        BharatMLClient client = new BharatMLClient("online-feature-store-api-mp.prd.meesho.int", 80, metadata);

        try {
            Query request = Query.newBuilder()
                    .setEntityLabel("catalog")
                    .addFeatureGroups(
                            FeatureGroup.newBuilder()
                                    .setLabel("derived_2_fp32")
                                    .addFeatureLabels("sbid_value")
                                    .build()
                    )
                    .addFeatureGroups(
                            FeatureGroup.newBuilder()
                                    .setLabel("derived_fp16")
                                    .addFeatureLabels("search__organic_clicks_by_views_3_days_percentile")
                                    .addFeatureLabels("search__organic_clicks_by_views_5_days_percentile")
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
            System.out.println("Result: " + result);

        } finally {
            try {
                client.shutdown();
            } catch (InterruptedException e) {
                e.printStackTrace();
            }
        }
    }
}
