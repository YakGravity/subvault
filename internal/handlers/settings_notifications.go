package handlers

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"subtrackr/internal/models"
	"subtrackr/internal/service"

	"github.com/gin-gonic/gin"
)

// SaveSMTPSettings saves SMTP configuration
func (h *SettingsHandler) SaveSMTPSettings(c *gin.Context) {
	var config models.SMTPConfig

	// Parse form data
	config.Host = c.PostForm("smtp_host")
	config.Username = c.PostForm("smtp_username")
	config.Password = c.PostForm("smtp_password")
	config.From = c.PostForm("smtp_from")
	config.FromName = c.PostForm("smtp_from_name")
	config.To = c.PostForm("smtp_to")

	// Parse port
	if portStr := c.PostForm("smtp_port"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Port = port
		}
	}

	// Validate required fields
	if config.Host == "" || config.Port == 0 || config.Username == "" || config.Password == "" || config.From == "" || config.To == "" {
		c.HTML(http.StatusBadRequest, "smtp-message.html", gin.H{
			"Error": tr(c, "settings_error_smtp_required", "Required SMTP fields: Host, Port, Username, Password, From email, To email"),
			"Type":  "error",
		})
		return
	}

	// Save configuration
	err := h.service.SaveSMTPConfig(&config)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "smtp-message.html", gin.H{
			"Error": err.Error(),
			"Type":  "error",
		})
		return
	}

	c.HTML(http.StatusOK, "smtp-message.html", gin.H{
		"Message": tr(c, "settings_success_smtp_saved", "SMTP settings saved successfully"),
		"Type":    "success",
	})
}

// TestSMTPConnection tests SMTP configuration with TLS/SSL support
func (h *SettingsHandler) TestSMTPConnection(c *gin.Context) {
	var config models.SMTPConfig

	// Parse form data
	config.Host = c.PostForm("smtp_host")
	config.Username = c.PostForm("smtp_username")
	config.Password = c.PostForm("smtp_password")
	config.From = c.PostForm("smtp_from")
	config.FromName = c.PostForm("smtp_from_name")
	config.To = c.PostForm("smtp_to")

	// Parse port
	if portStr := c.PostForm("smtp_port"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Port = port
		}
	}

	// Validate required fields for testing (connection test doesn't need From/To, but we validate for consistency)
	if config.Host == "" || config.Port == 0 || config.Username == "" || config.Password == "" {
		c.HTML(http.StatusBadRequest, "smtp-message.html", gin.H{
			"Error": tr(c, "settings_error_smtp_test_required", "Host, Port, Username, and Password are required for testing"),
			"Type":  "error",
		})
		return
	}

	// Test connection with TLS/SSL support
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	// Determine if this is an implicit TLS port (SMTPS)
	isSSLPort := config.Port == 465 || config.Port == 8465 || config.Port == 443

	var client *smtp.Client
	var err error

	if isSSLPort {
		// Use implicit TLS (direct SSL connection)
		tlsConfig := &tls.Config{
			ServerName: config.Host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			c.HTML(http.StatusBadRequest, "smtp-message.html", gin.H{
				"Error": fmt.Sprintf("Failed to connect via SSL: %v", err),
				"Type":  "error",
			})
			return
		}

		client, err = smtp.NewClient(conn, config.Host)
		if err != nil {
			conn.Close()
			c.HTML(http.StatusBadRequest, "smtp-message.html", gin.H{
				"Error": fmt.Sprintf("Failed to create SMTP client: %v", err),
				"Type":  "error",
			})
			return
		}
	} else {
		// Use STARTTLS (opportunistic TLS)
		client, err = smtp.Dial(addr)
		if err != nil {
			c.HTML(http.StatusBadRequest, "smtp-message.html", gin.H{
				"Error": fmt.Sprintf("Failed to connect: %v", err),
				"Type":  "error",
			})
			return
		}

		// Upgrade to TLS
		tlsConfig := &tls.Config{
			ServerName: config.Host,
		}

		if err = client.StartTLS(tlsConfig); err != nil {
			client.Close()
			c.HTML(http.StatusBadRequest, "smtp-message.html", gin.H{
				"Error": fmt.Sprintf("Failed to start TLS: %v", err),
				"Type":  "error",
			})
			return
		}
	}

	defer client.Close()

	// Try to authenticate
	if err = client.Auth(auth); err != nil {
		c.HTML(http.StatusBadRequest, "smtp-message.html", gin.H{
			"Error": fmt.Sprintf("Authentication failed: %v", err),
			"Type":  "error",
		})
		return
	}

	c.HTML(http.StatusOK, "smtp-message.html", gin.H{
		"Message": tr(c, "settings_success_smtp_test", "SMTP connection test successful!"),
		"Type":    "success",
	})
}

// GetSMTPConfig returns current SMTP configuration (without password)
func (h *SettingsHandler) GetSMTPConfig(c *gin.Context) {
	config, err := h.service.GetSMTPConfig()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"configured": false})
		return
	}

	// Don't send the password
	config.Password = ""
	c.JSON(http.StatusOK, gin.H{
		"configured": true,
		"config":     config,
	})
}

// SaveShoutrrrSettings saves Shoutrrr notification URL configuration
func (h *SettingsHandler) SaveShoutrrrSettings(c *gin.Context) {
	urlsRaw := c.PostForm("shoutrrr_urls")

	// Parse URLs (one per line)
	var urls []string
	for _, line := range strings.Split(urlsRaw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			urls = append(urls, line)
		}
	}

	if len(urls) == 0 {
		c.HTML(http.StatusBadRequest, "smtp-message.html", gin.H{
			"Error": tr(c, "settings_error_shoutrrr_required", "At least one notification URL is required"),
			"Type":  "error",
		})
		return
	}

	config := &models.ShoutrrrConfig{URLs: urls}
	err := h.service.SaveShoutrrrConfig(config)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "smtp-message.html", gin.H{
			"Error": err.Error(),
			"Type":  "error",
		})
		return
	}

	c.HTML(http.StatusOK, "smtp-message.html", gin.H{
		"Message": tr(c, "settings_success_shoutrrr_saved", "Notification settings saved successfully"),
		"Type":    "success",
	})
}

// TestShoutrrrConnection tests Shoutrrr notification URLs
func (h *SettingsHandler) TestShoutrrrConnection(c *gin.Context) {
	urlsRaw := c.PostForm("shoutrrr_urls")

	// Parse URLs (one per line)
	var urls []string
	for _, line := range strings.Split(urlsRaw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			urls = append(urls, line)
		}
	}

	if len(urls) == 0 {
		c.HTML(http.StatusBadRequest, "smtp-message.html", gin.H{
			"Error": tr(c, "settings_error_shoutrrr_test_required", "At least one notification URL is required for testing"),
			"Type":  "error",
		})
		return
	}

	// Test directly with the provided URLs (no need to save first)
	shoutrrrService := service.NewShoutrrrService(h.service)
	err := shoutrrrService.SendTestNotification(urls)
	if err != nil {
		c.HTML(http.StatusBadRequest, "smtp-message.html", gin.H{
			"Error": fmt.Sprintf("%s: %v", tr(c, "settings_error_shoutrrr_test_failed", "Failed to send test notification"), err),
			"Type":  "error",
		})
		return
	}

	c.HTML(http.StatusOK, "smtp-message.html", gin.H{
		"Message": tr(c, "settings_success_shoutrrr_test", "Test notification sent successfully! Check your devices."),
		"Type":    "success",
	})
}

// GetShoutrrrConfig returns current Shoutrrr configuration
func (h *SettingsHandler) GetShoutrrrConfig(c *gin.Context) {
	config, err := h.service.GetShoutrrrConfig()
	if err != nil || len(config.URLs) == 0 {
		c.JSON(http.StatusOK, gin.H{"configured": false, "url_count": 0})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configured": true,
		"url_count":  len(config.URLs),
	})
}

// UpdateNotificationSetting updates a notification preference
func (h *SettingsHandler) UpdateNotificationSetting(c *gin.Context) {
	setting := c.Param("setting")

	switch setting {
	case "renewal":
		enabled := !h.service.GetBoolSettingWithDefault("renewal_reminders", false)
		h.service.SetBoolSetting("renewal_reminders", enabled)
		c.JSON(http.StatusOK, gin.H{"enabled": enabled})
		return

	case "cancellation":
		enabled := !h.service.GetBoolSettingWithDefault("cancellation_reminders", false)
		h.service.SetBoolSetting("cancellation_reminders", enabled)
		c.JSON(http.StatusOK, gin.H{"enabled": enabled})
		return

	case "highcost":
		enabled := !h.service.GetBoolSettingWithDefault("high_cost_alerts", true)
		h.service.SetBoolSetting("high_cost_alerts", enabled)
		c.JSON(http.StatusOK, gin.H{"enabled": enabled})
		return

	case "reminder_days":
		daysStr := c.PostForm("reminder_days")
		if days, err := strconv.Atoi(daysStr); err == nil && days >= 1 && days <= 90 {
			h.service.SetIntSetting("reminder_days", days)
			c.JSON(http.StatusOK, gin.H{"days": days})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reminder days"})
		}
		return

	case "cancellation_reminder_days":
		daysStr := c.PostForm("cancellation_reminder_days")
		if days, err := strconv.Atoi(daysStr); err == nil && days >= 1 && days <= 90 {
			h.service.SetIntSetting("cancellation_reminder_days", days)
			c.JSON(http.StatusOK, gin.H{"days": days})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cancellation reminder days"})
		}
		return

	case "threshold":
		thresholdStr := c.PostForm("high_cost_threshold")
		if threshold, err := strconv.ParseFloat(thresholdStr, 64); err == nil && threshold >= 0 && threshold <= 10000 {
			err := h.service.SetFloatSetting("high_cost_threshold", threshold)
			if err != nil {
				slog.Error("failed to save high cost threshold", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"threshold": threshold})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid threshold value (must be between 0 and 10000)"})
		}

	case "budget":
		value := c.PostForm("value")
		if value == "" {
			value = "0"
		}
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid budget value"})
			return
		}
		if err := h.service.SetFloatSetting("monthly_budget", floatVal); err != nil {
			slog.Error("failed to save monthly budget", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
		return

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown setting"})
	}
}

// GetNotificationSettings returns current notification settings
func (h *SettingsHandler) GetNotificationSettings(c *gin.Context) {
	settings := models.NotificationSettings{
		RenewalReminders:         h.service.GetBoolSettingWithDefault("renewal_reminders", false),
		HighCostAlerts:           h.service.GetBoolSettingWithDefault("high_cost_alerts", true),
		HighCostThreshold:        h.service.GetFloatSettingWithDefault("high_cost_threshold", 50.0),
		ReminderDays:             h.service.GetIntSettingWithDefault("reminder_days", 7),
		CancellationReminders:    h.service.GetBoolSettingWithDefault("cancellation_reminders", false),
		CancellationReminderDays: h.service.GetIntSettingWithDefault("cancellation_reminder_days", 7),
	}

	c.JSON(http.StatusOK, settings)
}
