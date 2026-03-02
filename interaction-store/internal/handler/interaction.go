package handler

import (
	"context"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler/persist"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler/retrieve"
	"github.com/Meesho/BharatMLStack/interaction-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/interaction-store/pkg/proto/timeseries"
	"golang.org/x/sync/errgroup"
)

type InteractionHandler struct {
	timeseries.InteractionStoreTimeSeriesServiceServer
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
	start := time.Now()
	events := make([]model.ClickEvent, len(req.Data))
	for i, data := range req.Data {
		events[i] = model.ClickEvent{
			ClickEventData: model.ClickEventData{
				Payload: model.ClickEventPayload{
					UserId:    req.UserId,
					CatalogId: data.CatalogId,
					ProductId: data.ProductId,
					ClickedAt: data.Timestamp,
					Metadata:  data.Metadata,
				},
			},
		}
	}
	if err := h.clickPersistHandler.Persist(req.UserId, events); err != nil {
		metric.Timing("click_persist_latency", time.Since(start), []string{metric.TagAsString("status", "failure")})
		return nil, err
	}
	metric.Timing("click_persist_latency", time.Since(start), []string{metric.TagAsString("status", "success")})
	return &timeseries.PersistDataResponse{Message: "success"}, nil
}

func (h *InteractionHandler) PersistOrderData(ctx context.Context, req *timeseries.PersistOrderDataRequest) (*timeseries.PersistDataResponse, error) {
	start := time.Now()
	events := make([]model.FlattenedOrderEvent, len(req.Data))
	for i, data := range req.Data {
		events[i] = model.FlattenedOrderEvent{
			CatalogID:   data.CatalogId,
			ProductID:   data.ProductId,
			SubOrderNum: data.SubOrderNum,
			OrderedAt:   data.Timestamp,
			Metadata:    data.Metadata,
		}
	}
	if err := h.orderPersistHandler.Persist(req.UserId, events); err != nil {
		metric.Timing("order_persist_latency", time.Since(start), []string{metric.TagAsString("status", "failure")})
		return nil, err
	}
	metric.Timing("order_persist_latency", time.Since(start), []string{metric.TagAsString("status", "success")})
	return &timeseries.PersistDataResponse{Message: "success"}, nil
}

func (h *InteractionHandler) RetrieveClickInteractions(ctx context.Context, req *timeseries.RetrieveDataRequest) (*timeseries.RetrieveClickDataResponse, error) {
	start := time.Now()
	events, err := h.clickRetrieveHandler.Retrieve(req.UserId, req.StartTimestamp, req.EndTimestamp, req.Limit)
	if err != nil {
		metric.Timing("click_retrieve_latency", time.Since(start), []string{metric.TagAsString("status", "failure")})
		return nil, err
	}
	protoEvents := h.convertClickEventsToProto(events)
	metric.Timing("click_retrieve_latency", time.Since(start), []string{metric.TagAsString("status", "success")})
	return &timeseries.RetrieveClickDataResponse{Data: protoEvents}, nil
}

func (h *InteractionHandler) RetrieveOrderInteractions(ctx context.Context, req *timeseries.RetrieveDataRequest) (*timeseries.RetrieveOrderDataResponse, error) {
	start := time.Now()
	events, err := h.orderRetrieveHandler.Retrieve(req.UserId, req.StartTimestamp, req.EndTimestamp, req.Limit)
	if err != nil {
		metric.Timing("order_retrieve_latency", time.Since(start), []string{metric.TagAsString("status", "failure")})
		return nil, err
	}
	protoEvents := h.convertOrderEventsToProto(events)
	metric.Timing("order_retrieve_latency", time.Since(start), []string{metric.TagAsString("status", "success")})
	return &timeseries.RetrieveOrderDataResponse{Data: protoEvents}, nil
}

func (h *InteractionHandler) RetrieveInteractions(ctx context.Context, req *timeseries.RetrieveInteractionsRequest) (*timeseries.RetrieveInteractionsResponse, error) {
	start := time.Now()
	response := &timeseries.RetrieveInteractionsResponse{
		Data: make(map[string]*timeseries.InteractionData),
	}
	interactionData := &timeseries.InteractionData{}

	var wantClick, wantOrder bool
	for _, t := range req.InteractionTypes {
		switch t {
		case timeseries.InteractionTypeProto_CLICK:
			wantClick = true
		case timeseries.InteractionTypeProto_ORDER:
			wantOrder = true
		}
	}

	if wantClick && wantOrder {
		var clickProto []*timeseries.ClickEvent
		var orderProto []*timeseries.OrderEvent
		g := new(errgroup.Group)
		if wantClick {
			g.Go(func() error {
				events, err := h.clickRetrieveHandler.Retrieve(req.UserId, req.StartTimestamp, req.EndTimestamp, req.Limit)
				if err != nil {
					return err
				}
				clickProto = h.convertClickEventsToProto(events)
				return nil
			})
		}
		if wantOrder {
			g.Go(func() error {
				events, err := h.orderRetrieveHandler.Retrieve(req.UserId, req.StartTimestamp, req.EndTimestamp, req.Limit)
				if err != nil {
					return err
				}
				orderProto = h.convertOrderEventsToProto(events)
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			metric.Timing("interactions_retrieve_latency", time.Since(start), []string{metric.TagAsString("status", "failure"), metric.TagAsString("path", "parallel")})
			return nil, err
		}
		interactionData.ClickEvents = clickProto
		interactionData.OrderEvents = orderProto
		metric.Timing("interactions_retrieve_latency", time.Since(start), []string{metric.TagAsString("status", "success"), metric.TagAsString("path", "parallel")})
		response.Data[req.UserId] = interactionData
		return response, nil
	}

	if wantClick {
		events, err := h.clickRetrieveHandler.Retrieve(req.UserId, req.StartTimestamp, req.EndTimestamp, req.Limit)
		if err != nil {
			metric.Timing("interactions_retrieve_latency", time.Since(start), []string{metric.TagAsString("status", "failure"), metric.TagAsString("path", "sequential")})
			return nil, err
		}
		interactionData.ClickEvents = h.convertClickEventsToProto(events)
	}
	if wantOrder {
		events, err := h.orderRetrieveHandler.Retrieve(req.UserId, req.StartTimestamp, req.EndTimestamp, req.Limit)
		if err != nil {
			metric.Timing("interactions_retrieve_latency", time.Since(start), []string{metric.TagAsString("status", "failure"), metric.TagAsString("path", "sequential")})
			return nil, err
		}
		interactionData.OrderEvents = h.convertOrderEventsToProto(events)
	}
	response.Data[req.UserId] = interactionData
	metric.Timing("interactions_retrieve_latency", time.Since(start), []string{metric.TagAsString("status", "success"), metric.TagAsString("path", "sequential")})
	return response, nil
}

func (h *InteractionHandler) convertClickEventsToProto(events []model.ClickEvent) []*timeseries.ClickEvent {
	n := len(events)
	backing := make([]timeseries.ClickEvent, n)
	ptrs := make([]*timeseries.ClickEvent, n)
	for i := range events {
		p := &events[i].ClickEventData.Payload
		backing[i] = timeseries.ClickEvent{
			CatalogId: p.CatalogId,
			ProductId: p.ProductId,
			Timestamp: p.ClickedAt,
			Metadata:  p.Metadata,
		}
		ptrs[i] = &backing[i]
	}
	return ptrs
}

func (h *InteractionHandler) convertOrderEventsToProto(events []model.FlattenedOrderEvent) []*timeseries.OrderEvent {
	n := len(events)
	backing := make([]timeseries.OrderEvent, n)
	ptrs := make([]*timeseries.OrderEvent, n)
	for i := range events {
		e := &events[i]
		backing[i] = timeseries.OrderEvent{
			CatalogId:   e.CatalogID,
			ProductId:   e.ProductID,
			SubOrderNum: e.SubOrderNum,
			Timestamp:   e.OrderedAt,
			Metadata:    e.Metadata,
		}
		ptrs[i] = &backing[i]
	}
	return ptrs
}
