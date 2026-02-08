package handlers

import (
	"subtrackr/internal/service"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	settings    service.SettingsServiceInterface
	auth        service.AuthServiceInterface
	apiKey      service.APIKeyServiceInterface
	preferences service.PreferencesServiceInterface
	notifConfig service.NotificationConfigServiceInterface
	calendar    service.CalendarServiceInterface
	currency    service.CurrencyServiceInterface
}

func NewSettingsHandler(settings service.SettingsServiceInterface, auth service.AuthServiceInterface, apiKey service.APIKeyServiceInterface, preferences service.PreferencesServiceInterface, notifConfig service.NotificationConfigServiceInterface, calendar service.CalendarServiceInterface, currency service.CurrencyServiceInterface) *SettingsHandler {
	return &SettingsHandler{
		settings:    settings,
		auth:        auth,
		apiKey:      apiKey,
		preferences: preferences,
		notifConfig: notifConfig,
		calendar:    calendar,
		currency:    currency,
	}
}

// settingsBaseData returns common template data for all settings pages
func (h *SettingsHandler) settingsBaseData(c *gin.Context, currentTab string) gin.H {
	data := baseTemplateData(c)
	mergeTemplateData(data, gin.H{
		"CurrentPage": "settings",
		"CurrentTab":  currentTab,
	})
	return data
}
