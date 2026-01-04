package click

import "github.com/Meesho/interaction-store/internal/data/model"

type Consumer interface {
	Process(events []model.ClickEvent) error
}
