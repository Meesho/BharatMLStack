package com.bharatml.orchestrator.services.impls;

import com.bharatml.orchestrator.adaptor.ProtoConvertor;
import com.bharatml.orchestrator.dtos.*;
import com.bharatml.orchestrator.grpc.config.OnfsClientConfig;
import com.bharatml.orchestrator.grpc.service.FeatureServiceGrpc;
import com.bharatml.orchestrator.grpc.service.RetrieveProto;
import io.grpc.ManagedChannel;
import io.grpc.Metadata;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.MockedStatic;
import org.mockito.junit.jupiter.MockitoExtension;

import java.util.Arrays;
import java.util.Collections;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.Mockito.mockStatic;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class OnfsServiceTest {

    @Mock
    private ManagedChannel managedChannel;

    @Mock
    private FeatureServiceGrpc.FeatureServiceBlockingStub stub;

    @Mock
    private OnfsClientConfig onfsClientConfig;

    @Mock
    private OnfsClientConfig.Http2Config http2Config;

    private OnfsService onfsService;

    @BeforeEach
    void setUp() {
        when(onfsClientConfig.getHttp2Config()).thenReturn(http2Config);
        when(http2Config.getGrpcDeadline()).thenReturn(75);
        onfsService = new OnfsService(managedChannel, onfsClientConfig);
    }

    @Test
    void testRetrieveDecodedResult_withMetadata() {
        Query query = Query.builder()
                .entityLabel("sub_order")
                .featureGroups(Arrays.asList(
                        FeatureGroup.builder().label("derived_fp32").featureLabels(Arrays.asList("urs_v1_prob", "trs_v0_prob")).build(),
                        FeatureGroup.builder().label("derived_bool").featureLabels(Collections.singletonList("is_hold_out")).build(),
                        FeatureGroup.builder().label("derived_string").featureLabels(Arrays.asList("ensemble_v1_label", "trs_v0_label")).build()
                ))
                .keysSchema(Collections.singletonList("sub_order_num"))
                .keys(Collections.singletonList(Keys.builder().cols(Collections.singletonList("160816835296581632_1")).build()))
                .build();

        Metadata metadata = new Metadata();

        RetrieveProto.Query dummyQueryProto = RetrieveProto.Query.newBuilder().build();
        RetrieveProto.DecodedResult dummyGrpcResponse = RetrieveProto.DecodedResult.newBuilder().build();

        DecodedResult expectedResult = DecodedResult.builder()
                .keysSchema(Collections.singletonList("sub_order_num"))
                .featureSchemas(Arrays.asList(
                        FeatureSchema.builder().featureGroupLabel("derived_fp32").features(Arrays.asList(
                                Feature.builder().label("urs_v1_prob").columnIdx(0).build(),
                                Feature.builder().label("trs_v0_prob").columnIdx(1).build()
                        )).build(),
                        FeatureSchema.builder().featureGroupLabel("derived_bool").features(Collections.singletonList(
                                Feature.builder().label("is_hold_out").columnIdx(2).build()
                        )).build(),
                        FeatureSchema.builder().featureGroupLabel("derived_string").features(Arrays.asList(
                                Feature.builder().label("ensemble_v1_label").columnIdx(3).build(),
                                Feature.builder().label("trs_v0_label").columnIdx(4).build()
                        )).build()
                ))
                .rows(Collections.singletonList(
                        DecodedRow.builder()
                                .keys(Collections.singletonList("160816835296581632_1"))
                                .columns(Arrays.asList("0.27104712", "0.04461679", "true", "NO_QC-REDRESSAL", "NO_QC-REDRESSAL"))
                                .build()
                ))
                .build();

        try (
                MockedStatic<FeatureServiceGrpc> stubStatic = mockStatic(FeatureServiceGrpc.class);
                MockedStatic<ProtoConvertor> protoConvertorStatic = mockStatic(ProtoConvertor.class)
        ) {
            stubStatic.when(() -> FeatureServiceGrpc.newBlockingStub(managedChannel)).thenReturn(stub);
            protoConvertorStatic.when(() -> ProtoConvertor.convertToQueryProto(query)).thenReturn(dummyQueryProto);
            when(stub.withDeadlineAfter(anyLong(), any())).thenReturn(stub);
            when(stub.withInterceptors(any())).thenReturn(stub);
            when(stub.retrieveDecodedResult(dummyQueryProto)).thenReturn(dummyGrpcResponse);

            protoConvertorStatic.when(() -> ProtoConvertor.convertToDecodedResult(dummyGrpcResponse)).thenReturn(expectedResult);

            DecodedResult actualResult = onfsService.retrieveDecodedResult(metadata, query);
            assertEquals(expectedResult, actualResult);

            verify(stub).retrieveDecodedResult(dummyQueryProto);
        }
    }

}
