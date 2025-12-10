package com.bharatml.orchestrator.services;

import com.bharatml.orchestrator.dtos.DecodedResult;
import com.bharatml.orchestrator.dtos.Query;
import com.bharatml.orchestrator.dtos.Result;
import io.grpc.Metadata;

public interface IOnfsService {
    Result retrieveFeatures(Query request);
    Result retrieveFeatures(Metadata metadata, Query request);
    DecodedResult retrieveDecodedResult(Query request);
    DecodedResult retrieveDecodedResult(Metadata metadata, Query request);
} 