package com.bharatml.client.services;

import com.bharatml.client.dtos.DecodedResult;
import com.bharatml.client.dtos.Query;
import com.bharatml.client.dtos.Result;
import io.grpc.Metadata;

public interface IOnfsService {
    Result retrieveFeatures(Query request);
    Result retrieveFeatures(Metadata metadata, Query request);
    DecodedResult retrieveDecodedResult(Query request);
    DecodedResult retrieveDecodedResult(Metadata metadata, Query request);
} 