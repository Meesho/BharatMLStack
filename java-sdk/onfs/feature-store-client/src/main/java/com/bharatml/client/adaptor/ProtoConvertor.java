package com.bharatml.client.adaptor;

import com.bharatml.client.dtos.*;
import com.bharatml.client.grpc.service.RetrieveProto;

import java.util.List;
import java.util.stream.Collectors;

public class ProtoConvertor {

    public static RetrieveProto.Query convertToQueryProto(Query query) {
        List<RetrieveProto.FeatureGroup> featureGroups = query.getFeatureGroups().stream()
                .map(fg -> RetrieveProto.FeatureGroup.newBuilder()
                        .setLabel(fg.getLabel())
                        .addAllFeatureLabels(fg.getFeatureLabels())
                        .build())
                .collect(Collectors.toList());

        List<RetrieveProto.Keys> keys = query.getKeys().stream()
                .map(k -> RetrieveProto.Keys.newBuilder()
                        .addAllCols(k.getCols())
                        .build())
                .collect(Collectors.toList());

        return RetrieveProto.Query.newBuilder()
                .setEntityLabel(query.getEntityLabel())
                .addAllFeatureGroups(featureGroups)
                .addAllKeysSchema(query.getKeysSchema())
                .addAllKeys(keys)
                .build();
    }

    public static Result convertToResult(RetrieveProto.Result resultProto) {
        List<FeatureSchema> featureSchemas = resultProto.getFeatureSchemasList().stream()
                .map(fs -> FeatureSchema.builder()
                        .featureGroupLabel(fs.getFeatureGroupLabel())
                        .features(fs.getFeaturesList().stream()
                                .map(f -> Feature.builder()
                                        .label(f.getLabel())
                                        .columnIdx(f.getColumnIdx())
                                        .build())
                                .collect(Collectors.toList()))
                        .build())
                .collect(Collectors.toList());

        List<Row> rows = resultProto.getRowsList().stream()
                .map(r -> Row.builder()
                        .keys(r.getKeysList())
                        .columns(r.getColumnsList().stream().map(b -> b.toByteArray()).collect(Collectors.toList()))
                        .build())
                .collect(Collectors.toList());

        return Result.builder()
                .entityLabel(resultProto.getEntityLabel())
                .keysSchema(resultProto.getKeysSchemaList())
                .featureSchemas(featureSchemas)
                .rows(rows)
                .build();
    }


    public static DecodedResult convertToDecodedResult(RetrieveProto.DecodedResult decodedResultProto) {
        List<FeatureSchema> featureSchemas = decodedResultProto.getFeatureSchemasList().stream()
                .map(fs -> FeatureSchema.builder()
                        .featureGroupLabel(fs.getFeatureGroupLabel())
                        .features(fs.getFeaturesList().stream()
                                .map(f -> Feature.builder()
                                        .label(f.getLabel())
                                        .columnIdx(f.getColumnIdx())
                                        .build())
                                .collect(Collectors.toList()))
                        .build())
                .collect(Collectors.toList());

        List<DecodedRow> rows = decodedResultProto.getRowsList().stream()
                .map(r -> DecodedRow.builder()
                        .keys(r.getKeysList())
                        .columns(r.getColumnsList())
                        .build())
                .collect(Collectors.toList());

        return DecodedResult.builder()
                .keysSchema(decodedResultProto.getKeysSchemaList())
                .featureSchemas(featureSchemas)
                .rows(rows)
                .build();
    }
} 