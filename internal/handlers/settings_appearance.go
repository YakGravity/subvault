package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetTheme returns the current theme setting
func (h *SettingsHandler) GetTheme(c *gin.Context) {
	theme, err := h.preferences.GetTheme()
	if err != nil {
		// Default to 'default' theme if not set
		theme = "default"
	}

	c.JSON(http.StatusOK, gin.H{
		"theme": theme,
	})
}

// SetTheme saves the theme preference
func (h *SettingsHandler) SetTheme(c *gin.Context) {
	theme := c.PostForm("theme")

	// Validate theme mode
	validThemes := map[string]bool{
		"light":  true,
		"dark":   true,
		"system": true,
	}

	if !validThemes[theme] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid theme name",
		})
		return
	}

	if err := h.preferences.SetTheme(theme); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save theme",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"theme":   theme,
	})
}

// ToggleDarkMode toggles dark mode preference
func (h *SettingsHandler) ToggleDarkMode(c *gin.Context) {
	enabled := c.PostForm("enabled") == "true"

	err := h.preferences.SetDarkMode(enabled)
	if err != nil {
		slog.Error("failed to toggle dark mode", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dark_mode": enabled,
	})
}

// SetDateFormat handles POST /api/settings/date-format
func (h *SettingsHandler) SetDateFormat(c *gin.Context) {
	format := c.PostForm("format")

	validFormats := map[string]string{
		"DD.MM.YYYY": "02.01.2006",
		"MM/DD/YYYY": "01/02/2006",
		"YYYY-MM-DD": "2006-01-02",
		"":           "", // empty = locale default
	}

	goFormat, ok := validFormats[format]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
		return
	}

	if err := h.preferences.SetDateFormat(goFormat); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save date format"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "format": format})
}

// GetDateFormat handles GET /api/settings/date-format
func (h *SettingsHandler) GetDateFormat(c *gin.Context) {
	goFormat := h.preferences.GetDateFormat()

	// Map back to display format
	displayFormat := ""
	switch goFormat {
	case "02.01.2006":
		displayFormat = "DD.MM.YYYY"
	case "01/02/2006":
		displayFormat = "MM/DD/YYYY"
	case "2006-01-02":
		displayFormat = "YYYY-MM-DD"
	}

	c.JSON(http.StatusOK, gin.H{"format": displayFormat})
}
