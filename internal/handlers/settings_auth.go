package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupAuth enables authentication with username and password
func (h *SettingsHandler) SetupAuth(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")

	// Validate inputs
	if username == "" || password == "" {
		c.HTML(http.StatusBadRequest, "auth-message.html", gin.H{
			"Error": tr(c, "settings_error_auth_required", "Username and password are required"),
			"Type":  "error",
		})
		return
	}

	if password != confirmPassword {
		c.HTML(http.StatusBadRequest, "auth-message.html", gin.H{
			"Error": tr(c, "settings_error_password_mismatch", ErrPasswordsDoNotMatch),
			"Type":  "error",
		})
		return
	}

	if len(password) < 8 {
		c.HTML(http.StatusBadRequest, "auth-message.html", gin.H{
			"Error": tr(c, "settings_error_password_short", "Password must be at least 8 characters long"),
			"Type":  "error",
		})
		return
	}

	// Setup authentication
	err := h.auth.SetupAuth(username, password)
	if err != nil {
		slog.Error("failed to setup authentication", "error", err)
		c.HTML(http.StatusInternalServerError, "auth-message.html", gin.H{
			"Error": "An internal error occurred",
			"Type":  "error",
		})
		return
	}

	c.HTML(http.StatusOK, "auth-message.html", gin.H{
		"Message": tr(c, "settings_success_auth_enabled", "Authentication enabled successfully. You will need to login on next page load."),
		"Type":    "success",
	})
}

// DisableAuth disables authentication
func (h *SettingsHandler) DisableAuth(c *gin.Context) {
	err := h.auth.DisableAuth()
	if err != nil {
		slog.Error("failed to disable authentication", "error", err)
		c.HTML(http.StatusInternalServerError, "auth-message.html", gin.H{
			"Error": "An internal error occurred",
			"Type":  "error",
		})
		return
	}

	c.HTML(http.StatusOK, "auth-message.html", gin.H{
		"Message": tr(c, "settings_success_auth_disabled", "Authentication disabled successfully"),
		"Type":    "success",
	})
}

// GetAuthStatus returns the current authentication status
func (h *SettingsHandler) GetAuthStatus(c *gin.Context) {
	isEnabled := h.auth.IsAuthEnabled()
	username, _ := h.auth.GetAuthUsername()

	c.JSON(http.StatusOK, gin.H{
		"enabled":  isEnabled,
		"username": username,
	})
}
