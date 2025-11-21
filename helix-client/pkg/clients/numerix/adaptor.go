package numerix

import (
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/clients/numerix/client/grpc"
	"github.com/Meesho/go-core/datatypeconverter/typeconverter"
)

type IAdapter interface {
	MapRequestToProto(request *NumerixRequest) *grpc.NumerixRequestProto
	MapProtoToResponse(protoResponse *grpc.NumerixResponseProto) *NumerixResponse
}

type Adapter struct {
	IAdapter
}

func (a *Adapter) ScoreDataToFP32Bytes(request *NumerixRequest) {
	mapOfInputType := make(map[int]string)
	for i := range request.EntityScoreData.Schema {
		schemaParts := strings.Split(request.EntityScoreData.Schema[i], "@")
		if i == 0 {
			request.EntityScoreData.Schema[i] = schemaParts[0]
			continue
		}
		mapOfInputType[i] = schemaParts[1]
		request.EntityScoreData.Schema[i] = schemaParts[0]
	}

	for i := range request.EntityScoreData.Data {
		for j := range request.EntityScoreData.Data[i] {
			if j == 0 {
				continue
			}
			converted, err := typeconverter.ConvertBytesToBytes(request.EntityScoreData.Data[i][j], mapOfInputType[j], "datatypefp32")
			if err != nil {
				log.Error().Err(err).Msgf("Error converting bytes at index i=%d, j=%d, from type=%s to fp32", i, j, mapOfInputType[j])
				return
			}
			request.EntityScoreData.Data[i][j] = converted
		}
	}
}

func (a *Adapter) MapRequestToProto(request *NumerixRequest) *grpc.NumerixRequestProto {
	if request.EntityScoreData.Data == nil && request.EntityScoreData.StringData == nil {
		log.Warn().Msg("EntityScoreData Data and StringData is nil; cannot map request.")
		return nil
	}
	var (
		protoScores []*grpc.Score
		newSchema   []string
		dataType    = "fp32"
	)

	switch {
	case request.EntityScoreData.Data != nil:
		protoScores = make([]*grpc.Score, len(request.EntityScoreData.Data))
		a.ScoreDataToFP32Bytes(request)

		for rowIdx, row := range request.EntityScoreData.Data {
			protoScores[rowIdx] = &grpc.Score{
				MatrixFormat: &grpc.Score_ByteData{
					ByteData: &grpc.ByteList{Values: row},
				},
			}
		}
		newSchema = request.EntityScoreData.Schema

	case request.EntityScoreData.StringData != nil:
		protoScores = make([]*grpc.Score, len(request.EntityScoreData.StringData))
		for rowIdx, row := range request.EntityScoreData.StringData {
			protoScores[rowIdx] = &grpc.Score{
				MatrixFormat: &grpc.Score_StringData{
					StringData: &grpc.StringList{Values: row},
				},
			}
		}
		newSchema = request.EntityScoreData.Schema

	default:
		log.Warn().Msg("No valid score data found in EntityScoreData.")
		return nil
	}

	return &grpc.NumerixRequestProto{
		EntityScoreData: &grpc.EntityScoreData{
			Schema:       newSchema,
			EntityScores: protoScores,
			ComputeId:    request.EntityScoreData.ComputeID,
			DataType:     &dataType,
		},
	}
}

func (a *Adapter) MapProtoToResponse(protoResponse *grpc.NumerixResponseProto) *NumerixResponse {
	if errProto := protoResponse.GetError(); errProto != nil {
		log.Warn().Msgf("Received error in proto response error: %s", errProto.GetMessage())
		return &NumerixResponse{}
	}

	compDataProto := protoResponse.GetComputationScoreData()
	if compDataProto == nil {
		return &NumerixResponse{}
	}

	computationScores := compDataProto.GetComputationScores()

	inputSize := len(computationScores)
	data := make([][][]byte, inputSize)
	var stringData [][]string

	for i, scoreProto := range computationScores {
		switch T := scoreProto.GetMatrixFormat().(type) {
		case *grpc.Score_ByteData:
			if T.ByteData != nil && T.ByteData.GetValues() != nil {
				data[i] = T.ByteData.GetValues()
			}
		case *grpc.Score_StringData:
			if T.StringData != nil && T.StringData.GetValues() != nil {
				stringData = append(stringData, T.StringData.GetValues())
			}
		}
	}

	return &NumerixResponse{
		ComputationScoreData: ComputationScoreData{
			Schema:     compDataProto.GetSchema(),
			Data:       data,
			StringData: stringData,
		},
	}
}

func (a *Adapter) ConvertScoreDataFromFP32(data [][][]byte, dataType string) error {
	var err error

	for i := range data {
		for j := range data[i] {
			if data[i][j] == nil {
				continue
			}
			data[i][j], err = typeconverter.ConvertBytesToBytes(data[i][j], "fp32", dataType)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
