//go:build !meesho

package externalcall

import (
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
)

func InitSlackClient(SlackWebhookUrl string, SlackChannel string, SlackCcTags string, DefaultModelPath string) {
	log.Warn().Msg("Slack client is not supported without meesho build tag")
}

type SlackClient interface {
	SendCleanupNotification(deployableName string, deletedModels []string) error
	SendNumerixConfigCleanupNotification(deployableName string, deletedConfigs []string) error
	SendInferflowConfigCleanupNotification(deployableName string, deletedConfigs []string) error
}

type slackClientImpl struct {
	WebhookURL   string
	Channel      string
	BackupPath   string
	CCTags       string
	InactiveDays int
	HTTPClient   *http.Client
}

func GetSlackClient() SlackClient {
	log.Warn().Msg("Slack client is not supported without meesho build tag")
	return nil
}

type slackPayload struct {
	Username  string `json:"username"`
	IconEmoji string `json:"icon_emoji"`
	Channel   string `json:"channel"`
	Text      string `json:"text"`
}

func (s *slackClientImpl) SendCleanupNotification(deployableName string, deletedModels []string) error {
	return errors.New("Slack client is not supported without meesho build tag")
}

func (s *slackClientImpl) SendNumerixConfigCleanupNotification(deployableName string, deletedConfigs []string) error {
	return errors.New("Slack client is not supported without meesho build tag")
}

func (s *slackClientImpl) SendInferflowConfigCleanupNotification(deployableName string, deletedConfigs []string) error {
	return errors.New("Slack client is not supported without meesho build tag")
}
