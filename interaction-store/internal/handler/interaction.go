package handler

import (
	"context"
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler/persist"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler/retrieve"
	"github.com/Meesho/BharatMLStack/interaction-store/pkg/proto/timeseries"
)

type InteractionHandler struct {
	timeseries.TimeSeriesServiceServer
	clickPersistHandler  *persist.ClickPersistHandler
	clickRetrieveHandler *retrieve.ClickRetrieveHandler
	orderPersistHandler  *persist.OrderPersistHandler
	orderRetrieveHandler *retrieve.OrderRetrieveHandler
}

var (
	interactionHandler *InteractionHandler
	interactionOnce    sync.Once
)

func InitInteractionHandler() *InteractionHandler {
	interactionOnce.Do(func() {
		interactionHandler = &InteractionHandler{
			clickPersistHandler:  persist.InitClickPersistHandler(),
			clickRetrieveHandler: retrieve.InitClickRetrieveHandler(),
			orderPersistHandler:  persist.InitOrderPersistHandler(),
			orderRetrieveHandler: retrieve.InitOrderRetrieveHandler(),
		}
	})
	return interactionHandler
}

func (h *InteractionHandler) PersistClickData(ctx context.Context, req *timeseries.PersistClickDataRequest) (*timeseries.PersistDataResponse, error) {
	events := make([]model.ClickEvent, len(req.Data))
	for i, data := range req.Data {
		events[i] = model.ClickEvent{
			ClickEventData: model.ClickEventData{
				Payload: model.ClickEventPayload{
					UserId:    req.UserId,
					CatalogId: data.CatalogId,
					ProductId: data.ProductId,
					ClickedAt: data.Timestamp,
				},
			},
		}
	}
	if err := h.clickPersistHandler.Persist(req.UserId, events); err != nil {
		return nil, err
	}
	return &timeseries.PersistDataResponse{Message: "success"}, nil
}

func (h *InteractionHandler) PersistOrderData(ctx context.Context, req *timeseries.PersistOrderDataRequest) (*timeseries.PersistDataResponse, error) {
	events := make([]model.FlatOrderEvent, len(req.Data))
	for i, data := range req.Data {
		events[i] = model.FlatOrderEvent{
			CatalogID:   data.CatalogId,
			ProductID:   data.ProductId,
			SubOrderNum: data.SubOrderNum,
			OrderedAt:   data.Timestamp,
		}
	}
	if err := h.orderPersistHandler.Persist(req.UserId, events); err != nil {
		return nil, err
	}
	return &timeseries.PersistDataResponse{Message: "success"}, nil
}

func (h *InteractionHandler) RetrieveClickData(ctx context.Context, req *timeseries.RetrieveDataRequest) (*timeseries.RetrieveClickDataResponse, error) {
	data, err := h.clickRetrieveHandler.Retrieve(req.UserId, req.StartTimestamp, req.EndTimestamp, req.Limit)
	if err != nil {
		return nil, err
	}
	events := data.([]model.ClickEvent)
	protoEvents := make([]*timeseries.ClickEvent, len(events))
	for i, event := range events {
		protoEvents[i] = &timeseries.ClickEvent{
			UserId:    event.ClickEventData.Payload.UserId,
			CatalogId: event.ClickEventData.Payload.CatalogId,
			ProductId: event.ClickEventData.Payload.ProductId,
			Timestamp: event.ClickEventData.Payload.ClickedAt,
		}
	}
	return &timeseries.RetrieveClickDataResponse{Data: protoEvents}, nil
}

func (h *InteractionHandler) RetrieveOrderData(ctx context.Context, req *timeseries.RetrieveDataRequest) (*timeseries.RetrieveOrderDataResponse, error) {
	data, err := h.orderRetrieveHandler.Retrieve(req.UserId, req.StartTimestamp, req.EndTimestamp, req.Limit)
	if err != nil {
		return nil, err
	}
	events := data.([]model.FlatOrderEvent)
	protoEvents := make([]*timeseries.OrderEvent, len(events))
	for i, event := range events {
		protoEvents[i] = &timeseries.OrderEvent{
			CatalogId:   event.CatalogID,
			ProductId:   event.ProductID,
			SubOrderNum: event.SubOrderNum,
			Timestamp:   event.OrderedAt,
		}
	}
	return &timeseries.RetrieveOrderDataResponse{Data: protoEvents}, nil
}

func (h *InteractionHandler) RetrieveInteractions(ctx context.Context, req *timeseries.RetrieveInteractionsRequest) (*timeseries.RetrieveInteractionsResponse, error) {
	response := &timeseries.RetrieveInteractionsResponse{
		Data: make(map[string]*timeseries.InteractionData),
	}

	interactionData := &timeseries.InteractionData{}

	for _, interactionType := range req.InteractionTypes {
		switch interactionType {
		case timeseries.InteractionTypeProto_CLICK:
			data, err := h.clickRetrieveHandler.Retrieve(req.UserId, req.StartTimestamp, req.EndTimestamp, req.Limit)
			if err != nil {
				return nil, err
			}
			events := data.([]model.ClickEvent)
			protoEvents := make([]*timeseries.ClickEvent, len(events))
			for i, event := range events {
				protoEvents[i] = &timeseries.ClickEvent{
					UserId:    event.ClickEventData.Payload.UserId,
					CatalogId: event.ClickEventData.Payload.CatalogId,
					ProductId: event.ClickEventData.Payload.ProductId,
					Timestamp: event.ClickEventData.Payload.ClickedAt,
				}
			}
			interactionData.ClickEvents = protoEvents

		case timeseries.InteractionTypeProto_ORDER:
			data, err := h.orderRetrieveHandler.Retrieve(req.UserId, req.StartTimestamp, req.EndTimestamp, req.Limit)
			if err != nil {
				return nil, err
			}
			events := data.([]model.FlatOrderEvent)
			protoEvents := make([]*timeseries.OrderEvent, len(events))
			for i, event := range events {
				protoEvents[i] = &timeseries.OrderEvent{
					CatalogId:   event.CatalogID,
					ProductId:   event.ProductID,
					SubOrderNum: event.SubOrderNum,
					Timestamp:   event.OrderedAt,
				}
			}
			interactionData.OrderEvents = protoEvents
		}
	}

	response.Data[req.UserId] = interactionData
	return response, nil
}
