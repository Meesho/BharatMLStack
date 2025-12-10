package com.bharatml.orchestrator.dtos;

import lombok.Builder;
import lombok.Data;

import java.util.List;

@Data
@Builder
public class Keys {
    private List<String> cols;
} 