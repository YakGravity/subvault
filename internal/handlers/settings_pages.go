package handlers

import (
	"net/http"
	"subtrackr/internal/models"

	"github.com/gin-gonic/gin"
)

// SettingsGeneral renders the General settings page (Appearance, Language, Currency, About)
func (h *SettingsHandler) SettingsGeneral(c *gin.Context) {
	data := h.settingsBaseData(c, "general")
	mergeTemplateData(data, gin.H{
		"Title":    "Settings",
		"Currency": h.service.GetCurrency(),
		"Language": h.service.GetLanguage(),
	})
	c.HTML(http.StatusOK, "settings-general.html", data)
}

// SettingsAppearance renders the Appearance settings page (Light/Dark/System)
func (h *SettingsHandler) SettingsAppearance(c *gin.Context) {
	goFormat := h.service.GetDateFormat()
	displayFormat := ""
	switch goFormat {
	case "02.01.2006":
		displayFormat = "DD.MM.YYYY"
	case "01/02/2006":
		displayFormat = "MM/DD/YYYY"
	case "2006-01-02":
		displayFormat = "YYYY-MM-DD"
	}

	data := h.settingsBaseData(c, "appearance")
	mergeTemplateData(data, gin.H{
		"Title":      "Appearance",
		"DateFormat": displayFormat,
	})
	c.HTML(http.StatusOK, "settings-appearance.html", data)
}

// SettingsNotifications renders the Notifications settings page (SMTP, Shoutrrr, Preferences)
func (h *SettingsHandler) SettingsNotifications(c *gin.Context) {
	var smtpConfig *models.SMTPConfig
	smtpConfigured := false
	config, err := h.service.GetSMTPConfig()
	if err == nil && config != nil {
		config.Password = ""
		smtpConfig = config
		smtpConfigured = true
	}

	var shoutrrrConfig *models.ShoutrrrConfig
	shoutrrrConfigured := false
	shoutrrrCfg, err := h.service.GetShoutrrrConfig()
	if err == nil && shoutrrrCfg != nil && len(shoutrrrCfg.URLs) > 0 {
		shoutrrrConfig = shoutrrrCfg
		shoutrrrConfigured = true
	}

	data := h.settingsBaseData(c, "notifications")
	mergeTemplateData(data, gin.H{
		"Title":              "Notifications",
		"SMTPConfig":         smtpConfig,
		"SMTPConfigured":     smtpConfigured,
		"ShoutrrrConfig":     shoutrrrConfig,
		"ShoutrrrConfigured": shoutrrrConfigured,
		"CurrencySymbol":     h.service.GetCurrencySymbol(),
		"HighCostThreshold":  h.service.GetFloatSettingWithDefault("high_cost_threshold", 50.0),
		"MonthlyBudget":      h.service.GetFloatSettingWithDefault("monthly_budget", 0),
	})
	c.HTML(http.StatusOK, "settings-notifications.html", data)
}

// SettingsData renders the Data settings page (Export, Import, Backup, Calendar, Categories)
func (h *SettingsHandler) SettingsData(c *gin.Context) {
	calendarToken, _ := h.service.GetCalendarToken()

	data := h.settingsBaseData(c, "data")
	mergeTemplateData(data, gin.H{
		"Title":         "Data",
		"CalendarToken": calendarToken,
		"BaseURL":       "http://" + c.Request.Host,
	})
	c.HTML(http.StatusOK, "settings-data.html", data)
}

// SettingsSecurity renders the Security settings page (Auth, API Keys)
func (h *SettingsHandler) SettingsSecurity(c *gin.Context) {
	authEnabled := h.service.IsAuthEnabled()
	authUsername, _ := h.service.GetAuthUsername()

	var smtpConfigured bool
	_, err := h.service.GetSMTPConfig()
	if err == nil {
		smtpConfigured = true
	}

	data := h.settingsBaseData(c, "security")
	mergeTemplateData(data, gin.H{
		"Title":          "Security",
		"AuthEnabled":    authEnabled,
		"AuthUsername":   authUsername,
		"SMTPConfigured": smtpConfigured,
	})
	c.HTML(http.StatusOK, "settings-security.html", data)
}

// APIDocs renders the API documentation page
func (h *SettingsHandler) APIDocs(c *gin.Context) {
	data := h.settingsBaseData(c, "")
	mergeTemplateData(data, gin.H{
		"Title":       "API Documentation",
		"CurrentPage": "api-docs",
	})
	c.HTML(http.StatusOK, "api-docs.html", data)
}
