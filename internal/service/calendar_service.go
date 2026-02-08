package service

import (
	"crypto/rand"
	"fmt"
	"subtrackr/internal/repository"
)

type CalendarService struct {
	settings *SettingsService
	repo     *repository.SettingsRepository
}

func NewCalendarService(settings *SettingsService, repo *repository.SettingsRepository) *CalendarService {
	return &CalendarService{
		settings: settings,
		repo:     repo,
	}
}

// GenerateCalendarToken creates a new calendar feed token
func (c *CalendarService) GenerateCalendarToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := fmt.Sprintf("%x", bytes)
	if err := c.repo.Set(SettingKeyCalendarToken, token); err != nil {
		return "", err
	}
	c.settings.InvalidateCache()
	return token, nil
}

// GetCalendarToken retrieves the calendar feed token
func (c *CalendarService) GetCalendarToken() (string, error) {
	val, ok := c.settings.GetCached(SettingKeyCalendarToken)
	if !ok {
		return "", fmt.Errorf("calendar_token not found")
	}
	return val, nil
}

// RevokeCalendarToken deletes the calendar feed token
func (c *CalendarService) RevokeCalendarToken() error {
	defer c.settings.InvalidateCache()
	return c.repo.Set(SettingKeyCalendarToken, "")
}
