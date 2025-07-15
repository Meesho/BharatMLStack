package com.bharatmlstack.service;

import com.bharatmlstack.BharatMLClient;
import com.bharatmlstack.retrieve.Query;
import com.bharatmlstack.retrieve.FeatureGroup;
import com.bharatmlstack.retrieve.Keys;
import com.bharatmlstack.retrieve.Result;

import java.util.concurrent.TimeUnit;

public class Main {
    public static void main(String[] args) {
        System.out.println("Hello, feature-store-java!");

        BharatMLClient client = new BharatMLClient("localhost", 8081);

        try {
            Query request = Query.newBuilder()
                    .setEntityLabel("some_entity")
                    .addFeatureGroups(
                            FeatureGroup.newBuilder()
                                    .setLabel("fg_1")
                                    .addFeatureLabels("f1")
                                    .addFeatureLabels("f2")
                                    .build()
                    )
                    .addKeysSchema("key1")
                    .addKeys(
                            Keys.newBuilder().addCols("k1").build()
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