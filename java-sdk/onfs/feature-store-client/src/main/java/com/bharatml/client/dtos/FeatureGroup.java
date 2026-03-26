package com.bharatml.client.dtos;

import lombok.Builder;
import lombok.Data;

import java.util.List;

@Data
@Builder
public class FeatureGroup {
    private String label;
    private List<String> featureLabels;
} 