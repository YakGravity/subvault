package handlers

import (
	"log/slog"
	"net/http"

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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("HX-Refresh", "true")
	c.JSON(http.StatusOK, gin.H{"language": lang})
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
