package com.bharatml.client.dtos;

import lombok.Builder;
import lombok.Data;

@Data
@Builder
public class Feature {
    private String label;
    private int columnIdx;
} 