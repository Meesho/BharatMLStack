//go:build !meesho

package internal

import (
	"github.com/Meesho/BharatMLStack/interaction-store/internal/config"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/consumer/click"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/consumer/order"
)

func Init(config config.Configs) {
	click.Init(config)
	order.Init(config)
	// TODO: ADD KAFKA CONSUMER INITIALIZATION HERE
}
