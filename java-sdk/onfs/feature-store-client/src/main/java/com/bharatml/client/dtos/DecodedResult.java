package com.bharatml.client.dtos;

import lombok.Builder;
import lombok.Data;

import java.util.List;

@Data
@Builder
public class DecodedResult {
    private List<String> keysSchema;
    private List<FeatureSchema> featureSchemas;
    private List<DecodedRow> rows;
} 