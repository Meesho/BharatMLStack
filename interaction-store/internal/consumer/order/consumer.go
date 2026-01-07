package order

import "github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"

type Consumer interface {
	Process(event []model.OrderPlacedEvent) error
}
