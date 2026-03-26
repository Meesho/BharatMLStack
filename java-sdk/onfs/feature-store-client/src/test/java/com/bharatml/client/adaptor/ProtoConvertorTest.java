package com.bharatml.client.adaptor;

import com.bharatml.client.dtos.*;
import com.bharatml.client.grpc.service.RetrieveProto;
import org.junit.jupiter.api.Test;

import java.util.Collections;

import static org.junit.jupiter.api.Assertions.assertNotNull;

public class ProtoConvertorTest {
    @Test
    void testConvertToQueryProto() {
        Query query = Query.builder()
                .entityLabel("entity")
                .featureGroups(Collections.singletonList(
                        FeatureGroup.builder().label("fg").featureLabels(Collections.singletonList("f1")).build()
                ))
                .keysSchema(Collections.singletonList("key1"))
                .keys(Collections.singletonList(
                        Keys.builder().cols(Collections.singletonList("val1")).build()
                ))
                .build();
        RetrieveProto.Query proto = ProtoConvertor.convertToQueryProto(query);
        assertNotNull(proto);
    }

    @Test
    void testConvertToResult() {
        RetrieveProto.Result protoResult = RetrieveProto.Result.newBuilder()
                .setEntityLabel("entity")
                .addAllKeysSchema(Collections.singletonList("key1"))
                .addFeatureSchemas(
                        RetrieveProto.FeatureSchema.newBuilder()
                                .setFeatureGroupLabel("fg")
                                .addFeatures(RetrieveProto.Feature.newBuilder().setLabel("f1").setColumnIdx(0).build())
                                .build()
                )
                .addRows(
                        RetrieveProto.Row.newBuilder()
                                .addAllKeys(Collections.singletonList("val1"))
                                .addAllColumns(Collections.singletonList(com.google.protobuf.ByteString.copyFromUtf8("data")))
                                .build()
                )
                .build();
        Result result = ProtoConvertor.convertToResult(protoResult);
        assertNotNull(result);
    }

    @Test
    void testConvertToDecodedResult() {
        RetrieveProto.DecodedResult protoDecoded = RetrieveProto.DecodedResult.newBuilder()
                .addAllKeysSchema(Collections.singletonList("key1"))
                .addFeatureSchemas(
                        RetrieveProto.FeatureSchema.newBuilder()
                                .setFeatureGroupLabel("fg")
                                .addFeatures(RetrieveProto.Feature.newBuilder().setLabel("f1").setColumnIdx(0).build())
                                .build()
                )
                .addRows(
                        RetrieveProto.DecodedRow.newBuilder()
                                .addAllKeys(Collections.singletonList("val1"))
                                .addAllColumns(Collections.singletonList("col1"))
                                .build()
                )
                .build();
        DecodedResult result = ProtoConvertor.convertToDecodedResult(protoDecoded);
        assertNotNull(result);
    }
}
