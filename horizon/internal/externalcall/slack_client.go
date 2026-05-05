package externalcall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	initSlackOnce    sync.Once
	slackWebhookUrl  string
	slackChannel     string
	slackCcTags      string
	defaultModelPath string
)

func InitSlackClient(SlackWebhookUrl string, SlackChannel string, SlackCcTags string, DefaultModelPath string) {
	initSlackOnce.Do(func() {
		slackWebhookUrl = SlackWebhookUrl
		slackChannel = SlackChannel
		slackCcTags = SlackCcTags
		defaultModelPath = DefaultModelPath
	})
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

var (
	slackOnce     sync.Once
	slackInstance SlackClient
)

func GetSlackClient() SlackClient {
	slackOnce.Do(func() {
		slackInstance = &slackClientImpl{
			WebhookURL:   slackWebhookUrl,
			Channel:      slackChannel,
			BackupPath:   defaultModelPath,
			CCTags:       slackCcTags,
			InactiveDays: vmselectStartDaysAgo,
			HTTPClient: &http.Client{
				Timeout: 5 * time.Second,
			},
		}
	})
	return slackInstance
}

type slackPayload struct {
	Username  string `json:"username"`
	IconEmoji string `json:"icon_emoji"`
	Channel   string `json:"channel"`
	Text      string `json:"text"`
}

func (s *slackClientImpl) SendCleanupNotification(deployableName string, deletedModels []string) error {
	modelsText := strings.Join(deletedModels, "\n")
	ccTags := strings.Join(strings.Fields(s.CCTags), "\n") // handles space-separated tags
	message := fmt.Sprintf(`Prod Expt Models cleanup for %s
<!here>
Following models are deleted from prod experiment GCS repo and moved to backup folder (%s) as they aren't receiving any traffic in last %d days:
%s

cc:
%s`, deployableName, s.BackupPath, s.InactiveDays, modelsText, ccTags)

	payload := slackPayload{
		Username:  "Models Monitoring",
		IconEmoji: ":computer:",
		Channel:   s.Channel,
		Text:      message,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.WebhookURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}

func (s *slackClientImpl) SendNumerixConfigCleanupNotification(deployableName string, deletedConfigs []string) error {
	configsText := strings.Join(deletedConfigs, "\n")
	ccTags := strings.Join(strings.Fields(s.CCTags), "\n") // handles space-separated tags
	message := fmt.Sprintf(`Prod Numerix Config cleanup for %s
<!here>
Following Numerix configs are deleted from prod Numerix config registry as they aren't receiving any traffic in last %d days:
%s

cc:
%s`, deployableName, s.InactiveDays, configsText, ccTags)

	payload := slackPayload{
		Username:  "Numerix Config Monitoring",
		IconEmoji: ":computer:",
		Channel:   s.Channel,
		Text:      message,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.WebhookURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}

func (s *slackClientImpl) SendInferflowConfigCleanupNotification(deployableName string, deletedConfigs []string) error {
	configsText := strings.Join(deletedConfigs, "\n")
	ccTags := strings.Join(strings.Fields(s.CCTags), "\n") // handles space-separated tags
	message := fmt.Sprintf(`Prod Inferflow Config cleanup for %s
<!here>
Following Inferflow configs are deleted from prod Inferflow config registry as they aren't receiving any traffic in last %d days:
%s

cc:
%s`, deployableName, s.InactiveDays, configsText, ccTags)

	payload := slackPayload{
		Username:  "Inferflow Config Monitoring",
		IconEmoji: ":computer:",
		Channel:   s.Channel,
		Text:      message,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.WebhookURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}
