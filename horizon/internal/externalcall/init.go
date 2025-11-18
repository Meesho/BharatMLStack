package externalcall

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

func Init(config configs.Configs) {
	InitPrometheusClient(config.VmselectStartDaysAgo, config.VmselectBaseUrl, config.VmselectApiKey)
	InitSlackClient(config.SlackWebhookUrl, config.SlackChannel, config.SlackCcTags, config.DefaultModelPath)
	InitRingmasterClient(config.RingmasterBaseUrl, config.RingmasterMiscSession, config.RingmasterAuthorization, config.RingmasterEnvironment, config.RingmasterApiKey)
	// Initialize feature validation client with config-based URLs
	InitFeatureValidationClient(config)
	// Pricing client is initialized in main.go
}
