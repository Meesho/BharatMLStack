package click

import "github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"

type Consumer interface {
	Process(events []model.ClickEvent) error
}
