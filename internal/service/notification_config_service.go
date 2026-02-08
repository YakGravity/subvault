package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"subvault/internal/models"
	"subvault/internal/repository"
)

type NotificationConfigService struct {
	settings *SettingsService
	repo     *repository.SettingsRepository
}

func NewNotificationConfigService(settings *SettingsService, repo *repository.SettingsRepository) *NotificationConfigService {
	return &NotificationConfigService{
		settings: settings,
		repo:     repo,
	}
}

// SaveSMTPConfig saves SMTP configuration
func (n *NotificationConfigService) SaveSMTPConfig(config *models.SMTPConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	defer n.settings.InvalidateCache()
	return n.repo.Set(SettingKeySMTPConfig, string(data))
}

// GetSMTPConfig retrieves SMTP configuration
func (n *NotificationConfigService) GetSMTPConfig() (*models.SMTPConfig, error) {
	data, ok := n.settings.GetCached(SettingKeySMTPConfig)
	if !ok {
		return nil, fmt.Errorf("smtp_config not found")
	}

	var config models.SMTPConfig
	err := json.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveShoutrrrConfig saves Shoutrrr configuration
func (n *NotificationConfigService) SaveShoutrrrConfig(config *models.ShoutrrrConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	defer n.settings.InvalidateCache()
	return n.repo.Set(SettingKeyShoutrrrConfig, string(data))
}

// GetShoutrrrConfig retrieves Shoutrrr configuration
func (n *NotificationConfigService) GetShoutrrrConfig() (*models.ShoutrrrConfig, error) {
	data, ok := n.settings.GetCached(SettingKeyShoutrrrConfig)
	if !ok {
		return nil, fmt.Errorf("shoutrrr_config not found")
	}

	var config models.ShoutrrrConfig
	err := json.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// MigratePushoverToShoutrrr migrates existing Pushover config to Shoutrrr format
func (n *NotificationConfigService) MigratePushoverToShoutrrr() error {
	data, ok := n.settings.GetCached(SettingKeyPushoverConfig)
	if !ok {
		return nil // No Pushover config exists, nothing to migrate
	}

	var oldConfig struct {
		UserKey  string `json:"pushover_user_key"`
		AppToken string `json:"pushover_app_token"`
	}
	if err := json.Unmarshal([]byte(data), &oldConfig); err != nil {
		return nil // Invalid config, skip migration
	}

	if oldConfig.UserKey == "" || oldConfig.AppToken == "" {
		return nil // Empty config, skip migration
	}

	// Check if Shoutrrr config already exists
	if existing, err := n.GetShoutrrrConfig(); err == nil && len(existing.URLs) > 0 {
		return nil // Already migrated
	}

	// Convert to Shoutrrr Pushover URL format
	shoutrrrURL := fmt.Sprintf("pushover://shoutrrr:%s@%s/", oldConfig.AppToken, oldConfig.UserKey)

	newConfig := &models.ShoutrrrConfig{
		URLs: []string{shoutrrrURL},
	}

	if err := n.SaveShoutrrrConfig(newConfig); err != nil {
		return fmt.Errorf("failed to save migrated Shoutrrr config: %w", err)
	}

	// Delete old Pushover config
	n.repo.Delete(SettingKeyPushoverConfig)
	slog.Info("migrated Pushover config to Shoutrrr URL format")

	return nil
}
