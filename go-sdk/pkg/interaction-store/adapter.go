package interactionstore

import (
	pb "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/interaction-store/timeseries"
)

// Adapter handles conversion between SDK models and proto messages
type Adapter struct{}

// ConvertToPersistClickDataProto converts SDK request to proto request
func (a *Adapter) ConvertToPersistClickDataProto(request *PersistClickDataRequest) *pb.PersistClickDataRequest {
	if request == nil {
		return nil
	}

	protoData := make([]*pb.ClickData, len(request.Data))
	for i, d := range request.Data {
		protoData[i] = &pb.ClickData{
			CatalogId: d.CatalogId,
			ProductId: d.ProductId,
			Timestamp: d.Timestamp,
			Metadata:  d.Metadata,
		}
	}

	return &pb.PersistClickDataRequest{
		UserId: request.UserId,
		Data:   protoData,
	}
}

// ConvertToPersistOrderDataProto converts SDK request to proto request
func (a *Adapter) ConvertToPersistOrderDataProto(request *PersistOrderDataRequest) *pb.PersistOrderDataRequest {
	if request == nil {
		return nil
	}

	protoData := make([]*pb.OrderData, len(request.Data))
	for i, d := range request.Data {
		protoData[i] = &pb.OrderData{
			CatalogId:   d.CatalogId,
			ProductId:   d.ProductId,
			SubOrderNum: d.SubOrderNum,
			Timestamp:   d.Timestamp,
			Metadata:    d.Metadata,
		}
	}

	return &pb.PersistOrderDataRequest{
		UserId: request.UserId,
		Data:   protoData,
	}
}

// ConvertToRetrieveDataProto converts SDK request to proto request
func (a *Adapter) ConvertToRetrieveDataProto(request *RetrieveDataRequest) *pb.RetrieveDataRequest {
	if request == nil {
		return nil
	}

	return &pb.RetrieveDataRequest{
		UserId:         request.UserId,
		StartTimestamp: request.StartTimestamp,
		EndTimestamp:   request.EndTimestamp,
		Limit:          request.Limit,
	}
}

// ConvertToRetrieveInteractionsProto converts SDK request to proto request
func (a *Adapter) ConvertToRetrieveInteractionsProto(request *RetrieveInteractionsRequest) *pb.RetrieveInteractionsRequest {
	if request == nil {
		return nil
	}

	interactionTypes := make([]pb.InteractionType, len(request.InteractionTypes))
	for i, t := range request.InteractionTypes {
		interactionTypes[i] = pb.InteractionType(t)
	}

	return &pb.RetrieveInteractionsRequest{
		UserId:           request.UserId,
		InteractionTypes: interactionTypes,
		StartTimestamp:   request.StartTimestamp,
		EndTimestamp:     request.EndTimestamp,
		Limit:            request.Limit,
	}
}

// ConvertFromPersistDataResponse converts proto response to SDK response
func (a *Adapter) ConvertFromPersistDataResponse(response *pb.PersistDataResponse) *PersistDataResponse {
	if response == nil {
		return nil
	}

	return &PersistDataResponse{
		Message: response.Message,
	}
}

// ConvertFromRetrieveClickDataResponse converts proto response to SDK response
func (a *Adapter) ConvertFromRetrieveClickDataResponse(response *pb.RetrieveClickDataResponse) *RetrieveClickDataResponse {
	if response == nil || len(response.Data) == 0 {
		return &RetrieveClickDataResponse{Data: []ClickEvent{}}
	}

	events := make([]ClickEvent, len(response.Data))
	for i, e := range response.Data {
		events[i] = ClickEvent{
			CatalogId: e.CatalogId,
			ProductId: e.ProductId,
			Timestamp: e.Timestamp,
			Metadata:  e.Metadata,
		}
	}

	return &RetrieveClickDataResponse{Data: events}
}

// ConvertFromRetrieveOrderDataResponse converts proto response to SDK response
func (a *Adapter) ConvertFromRetrieveOrderDataResponse(response *pb.RetrieveOrderDataResponse) *RetrieveOrderDataResponse {
	if response == nil || len(response.Data) == 0 {
		return &RetrieveOrderDataResponse{Data: []OrderEvent{}}
	}

	events := make([]OrderEvent, len(response.Data))
	for i, e := range response.Data {
		events[i] = OrderEvent{
			CatalogId:   e.CatalogId,
			ProductId:   e.ProductId,
			SubOrderNum: e.SubOrderNum,
			Timestamp:   e.Timestamp,
			Metadata:    e.Metadata,
		}
	}

	return &RetrieveOrderDataResponse{Data: events}
}

// ConvertFromRetrieveInteractionsResponse converts proto response to SDK response
func (a *Adapter) ConvertFromRetrieveInteractionsResponse(response *pb.RetrieveInteractionsResponse) *RetrieveInteractionsResponse {
	if response == nil || len(response.Data) == 0 {
		return &RetrieveInteractionsResponse{Data: make(map[string]InteractionData)}
	}

	data := make(map[string]InteractionData)
	for key, val := range response.Data {
		clickEvents := make([]ClickEvent, len(val.ClickEvents))
		for i, e := range val.ClickEvents {
			clickEvents[i] = ClickEvent{
				CatalogId: e.CatalogId,
				ProductId: e.ProductId,
				Timestamp: e.Timestamp,
				Metadata:  e.Metadata,
			}
		}

		orderEvents := make([]OrderEvent, len(val.OrderEvents))
		for i, e := range val.OrderEvents {
			orderEvents[i] = OrderEvent{
				CatalogId:   e.CatalogId,
				ProductId:   e.ProductId,
				SubOrderNum: e.SubOrderNum,
				Timestamp:   e.Timestamp,
				Metadata:    e.Metadata,
			}
		}

		data[key] = InteractionData{
			ClickEvents: clickEvents,
			OrderEvents: orderEvents,
		}
	}

	return &RetrieveInteractionsResponse{Data: data}
}
