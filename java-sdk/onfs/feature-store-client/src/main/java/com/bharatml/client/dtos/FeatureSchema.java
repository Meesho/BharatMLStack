package com.bharatml.client.dtos;

import lombok.Builder;
import lombok.Data;

import java.util.List;

@Data
@Builder
public class FeatureSchema {
    private String featureGroupLabel;
    private List<Feature> features;
} 