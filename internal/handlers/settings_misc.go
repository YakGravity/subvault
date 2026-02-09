package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"subvault/internal/service"

	"github.com/gin-gonic/gin"
)

// UpdateCurrency updates the currency preference
func (h *SettingsHandler) UpdateCurrency(c *gin.Context) {
	currency := c.PostForm("currency")

	err := h.preferences.SetCurrency(currency)
	if err != nil {
		slog.Error("failed to set currency", "error", err)
		c.String(http.StatusBadRequest, "Invalid currency")
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateLanguage updates the language preference
func (h *SettingsHandler) UpdateLanguage(c *gin.Context) {
	lang := c.PostForm("language")

	err := h.preferences.SetLanguage(lang)
	if err != nil {
		slog.Error("failed to set language", "error", err)
		c.String(http.StatusBadRequest, "Invalid language")
		return
	}

	c.Header("HX-Refresh", "true")
	c.Status(http.StatusNoContent)
}

// GenerateCalendarToken creates a new calendar feed token
func (h *SettingsHandler) GenerateCalendarToken(c *gin.Context) {
	token, err := h.calendar.GenerateCalendarToken()
	if err != nil {
		slog.Error("failed to generate calendar token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":   token,
	})
}

// RevokeCalendarToken deletes the calendar feed token
func (h *SettingsHandler) RevokeCalendarToken(c *gin.Context) {
	if err := h.calendar.RevokeCalendarToken(); err != nil {
		slog.Error("failed to revoke calendar token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RefreshExchangeRates manually refreshes exchange rates from ECB
func (h *SettingsHandler) RefreshExchangeRates(c *gin.Context) {
	err := h.currency.RefreshRates()
	status := h.currency.GetStatus()

	data := baseTemplateData(c)
	mergeTemplateData(data, gin.H{
		"RateStatus": status,
	})

	if err != nil {
		slog.Warn("manual exchange rate refresh failed", "error", err)
		mergeTemplateData(data, gin.H{
			"RefreshError": true,
		})
	} else {
		mergeTemplateData(data, gin.H{
			"RefreshSuccess": true,
		})
	}

	c.HTML(http.StatusOK, "exchange-rate-status.html", data)
}

// UpdateCurrencyRefreshInterval updates the exchange rate refresh interval
func (h *SettingsHandler) UpdateCurrencyRefreshInterval(c *gin.Context) {
	hoursStr := c.PostForm("hours")
	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours < 1 || hours > 168 {
		c.String(http.StatusBadRequest, "Invalid interval (1-168 hours)")
		return
	}

	if err := h.settings.SetIntSetting(service.SettingKeyCurrencyRefreshHours, hours); err != nil {
		slog.Error("failed to save currency refresh interval", "error", err)
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}

	c.Status(http.StatusNoContent)
}
