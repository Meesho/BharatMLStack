package handler

import (
	"context"
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler/persist"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler/retrieve"
	"github.com/Meesho/BharatMLStack/interaction-store/pkg/proto/timeseries"
)

type InteractionHandler struct {
	timeseries.TimeSeriesServiceServer
	clickPersistHandler  *persist.ClickPersistHandler
	clickRetrieveHandler *retrieve.ClickRetrieveHandler
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
		}
	})
	return interactionHandler
}

func (h *InteractionHandler) PersistClickData(ctx context.Context, req *timeseries.PersistClickDataRequest) (*timeseries.PersistDataResponse, error) {
	return nil, nil
}

func (h *InteractionHandler) PersistOrderData(ctx context.Context, req *timeseries.PersistOrderDataRequest) (*timeseries.PersistDataResponse, error) {
	return nil, nil
}

func (h *InteractionHandler) RetrieveClickData(ctx context.Context, req *timeseries.RetrieveDataRequest) (*timeseries.RetrieveClickDataResponse, error) {
	return nil, nil
}

func (h *InteractionHandler) RetrieveOrderData(ctx context.Context, req *timeseries.RetrieveDataRequest) (*timeseries.RetrieveOrderDataResponse, error) {
	return nil, nil
}

func (h *InteractionHandler) RetrieveInteractions(ctx context.Context, req *timeseries.RetrieveInteractionsRequest) (*timeseries.RetrieveInteractionsResponse, error) {
	return nil, nil
}
