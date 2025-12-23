package com.bharatml.client.dtos;

import lombok.Builder;
import lombok.Data;

import java.util.List;

@Data
@Builder
public class Query {
    private String entityLabel;
    private List<FeatureGroup> featureGroups;
    private List<String> keysSchema;
    private List<Keys> keys;
} 