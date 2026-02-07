package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// UpdateCurrency updates the currency preference
func (h *SettingsHandler) UpdateCurrency(c *gin.Context) {
	currency := c.PostForm("currency")

	err := h.service.SetCurrency(currency)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"currency": currency,
		"symbol":   h.service.GetCurrencySymbol(),
	})
}

// UpdateLanguage updates the language preference
func (h *SettingsHandler) UpdateLanguage(c *gin.Context) {
	lang := c.PostForm("language")

	err := h.service.SetLanguage(lang)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("HX-Refresh", "true")
	c.JSON(http.StatusOK, gin.H{"language": lang})
}

// GenerateCalendarToken creates a new calendar feed token
func (h *SettingsHandler) GenerateCalendarToken(c *gin.Context) {
	token, err := h.service.GenerateCalendarToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":   token,
	})
}

// RevokeCalendarToken deletes the calendar feed token
func (h *SettingsHandler) RevokeCalendarToken(c *gin.Context) {
	if err := h.service.RevokeCalendarToken(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
