package handlers

import (
	"crypto/subtle"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"subvault/internal/crypto"
	"subvault/internal/models"

	"github.com/gin-gonic/gin"
)

// ExportCSV exports all subscriptions as CSV
func (h *SubscriptionHandler) ExportCSV(c *gin.Context) {
	subscriptions, err := h.service.GetAll()
	if err != nil {
		slog.Error("failed to get subscriptions for CSV export", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=subscriptions.csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write CSV header
	header := []string{"ID", "Name", "Category", "Cost", "Tax Rate", "Price Type", "Net Cost", "Gross Cost", "Tax Amount", "Schedule", "Status", "Payment Method", "Login Name", "Customer Number", "Contract Number", "Start Date", "Renewal Date", "Cancellation Date", "URL", "Notes", "Usage", "Renewal Reminder", "Renewal Reminder Days", "Cancellation Reminder", "Cancellation Reminder Days", "High Cost Alert", "Created At"}
	writer.Write(header)

	// Write subscription data
	for _, sub := range subscriptions {
		categoryName := ""
		if sub.Category.Name != "" {
			categoryName = sub.Category.Name
		}
		record := []string{
			fmt.Sprintf("%d", sub.ID),
			sub.Name,
			categoryName,
			fmt.Sprintf("%.2f", sub.Cost),
			fmt.Sprintf("%.2f", sub.TaxRate),
			sub.PriceType,
			fmt.Sprintf("%.2f", sub.NetCost()),
			fmt.Sprintf("%.2f", sub.GrossCost()),
			fmt.Sprintf("%.2f", sub.TaxAmount()),
			sub.Schedule,
			sub.Status,
			sub.PaymentMethod,
			sub.LoginName,
			sub.CustomerNumber,
			sub.ContractNumber,
			formatDate(sub.StartDate),
			formatDate(sub.RenewalDate),
			formatDate(sub.CancellationDate),
			sub.URL,
			sub.Notes,
			sub.Usage,
			fmt.Sprintf("%t", sub.RenewalReminder),
			fmt.Sprintf("%d", sub.RenewalReminderDays),
			fmt.Sprintf("%t", sub.CancellationReminder),
			fmt.Sprintf("%d", sub.CancellationReminderDays),
			fmt.Sprintf("%t", sub.HighCostAlert),
			sub.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		writer.Write(record)
	}
}

// ExportJSON exports all subscriptions as JSON
func (h *SubscriptionHandler) ExportJSON(c *gin.Context) {
	subscriptions, err := h.service.GetAll()
	if err != nil {
		slog.Error("failed to get subscriptions for JSON export", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=subscriptions.json")

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subscriptions,
		"exported_at":   time.Now(),
		"total_count":   len(subscriptions),
	})
}

// ExportEncrypted creates an AES-256-GCM encrypted backup file (.stbk)
func (h *SubscriptionHandler) ExportEncrypted(c *gin.Context) {
	password := c.PostForm("password")
	if password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrPasswordRequired})
		return
	}

	subscriptions, err := h.service.GetAll()
	if err != nil {
		slog.Error("failed to get subscriptions for encrypted export", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	categories, err := h.service.GetAllCategories()
	if err != nil {
		categories = nil
	}

	backupData := gin.H{
		"subscriptions": subscriptions,
		"categories":    categories,
		"exported_at":   time.Now(),
		"total_count":   len(subscriptions),
		"version":       "2.0",
	}

	jsonData, err := json.Marshal(backupData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize data"})
		return
	}

	encrypted, err := crypto.Encrypt(jsonData, password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Encryption failed"})
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="subvault-backup.stbk"`)
	c.Data(http.StatusOK, "application/octet-stream", encrypted)
}

// BackupData creates a complete backup of all data
func (h *SubscriptionHandler) BackupData(c *gin.Context) {
	subscriptions, err := h.service.GetAll()
	if err != nil {
		slog.Error("failed to get subscriptions for backup", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	stats, err := h.service.GetStats()
	if err != nil {
		slog.Error("failed to get stats for backup", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	backup := gin.H{
		"version":       "1.0",
		"backup_date":   time.Now(),
		"subscriptions": subscriptions,
		"stats":         stats,
		"total_count":   len(subscriptions),
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=subvault-backup.json")
	c.JSON(http.StatusOK, backup)
}

// ClearAllData removes all subscription data
func (h *SubscriptionHandler) ClearAllData(c *gin.Context) {
	subscriptions, err := h.service.GetAll()
	if err != nil {
		slog.Error("failed to get subscriptions for clearing data", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Delete all subscriptions
	for _, sub := range subscriptions {
		err := h.service.Delete(sub.ID)
		if err != nil {
			slog.Error("failed to delete subscription during clear", "error", err, "id", sub.ID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "All subscription data has been cleared",
		"deleted_count": len(subscriptions),
	})
}

// ExportICal generates and downloads an iCal file with all subscription renewal dates
func (h *SubscriptionHandler) ExportICal(c *gin.Context) {
	subscriptions, err := h.service.GetAll()
	if err != nil {
		slog.Error("failed to get subscriptions for iCal export", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	icalContent := h.generateICal(subscriptions)

	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="subvault-renewals.ics"`)
	c.Data(http.StatusOK, "text/calendar; charset=utf-8", []byte(icalContent))
}

// ServeCalendarFeed serves iCal data for calendar subscription via token
func (h *SubscriptionHandler) ServeCalendarFeed(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.Status(http.StatusNotFound)
		return
	}

	storedToken, err := h.calendarService.GetCalendarToken()
	if err != nil || storedToken == "" || subtle.ConstantTimeCompare([]byte(storedToken), []byte(token)) != 1 {
		c.Status(http.StatusNotFound)
		return
	}

	subscriptions, err := h.service.GetAll()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	icalContent := h.generateICal(subscriptions)

	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Data(http.StatusOK, "text/calendar; charset=utf-8", []byte(icalContent))
}

// generateICal creates iCal content from subscriptions
func (h *SubscriptionHandler) generateICal(subscriptions []models.Subscription) string {
	icalContent := "BEGIN:VCALENDAR\r\n"
	icalContent += "VERSION:2.0\r\n"
	icalContent += "PRODID:-//SubVault//Subscription Renewals//EN\r\n"
	icalContent += "CALSCALE:GREGORIAN\r\n"
	icalContent += "METHOD:PUBLISH\r\n"

	now := time.Now()
	for _, sub := range subscriptions {
		if sub.RenewalDate != nil && sub.Status == "Active" {
			dtStart := sub.RenewalDate.Format("20060102T150000Z")
			dtEnd := sub.RenewalDate.Add(1 * time.Hour).Format("20060102T150000Z")
			dtStamp := now.Format("20060102T150000Z")
			uid := fmt.Sprintf("subvault-%d-%d@subvault", sub.ID, sub.RenewalDate.Unix())

			summary := fmt.Sprintf("%s Renewal", sub.Name)
			description := fmt.Sprintf("Subscription: %s\\nCost: %s%.2f\\nSchedule: %s", sub.Name, h.preferences.GetCurrencySymbol(), sub.Cost, sub.Schedule)
			if sub.URL != "" {
				description += fmt.Sprintf("\\nURL: %s", sub.URL)
			}

			icalContent += "BEGIN:VEVENT\r\n"
			icalContent += fmt.Sprintf("UID:%s\r\n", uid)
			icalContent += fmt.Sprintf("DTSTAMP:%s\r\n", dtStamp)
			icalContent += fmt.Sprintf("DTSTART:%s\r\n", dtStart)
			icalContent += fmt.Sprintf("DTEND:%s\r\n", dtEnd)
			icalContent += fmt.Sprintf("SUMMARY:%s\r\n", summary)
			icalContent += fmt.Sprintf("DESCRIPTION:%s\r\n", description)
			icalContent += "STATUS:CONFIRMED\r\n"
			icalContent += "SEQUENCE:0\r\n"

			switch sub.Schedule {
			case "Daily":
				icalContent += "RRULE:FREQ=DAILY;INTERVAL=1\r\n"
			case "Weekly":
				icalContent += "RRULE:FREQ=WEEKLY;INTERVAL=1\r\n"
			case "Monthly":
				icalContent += "RRULE:FREQ=MONTHLY;INTERVAL=1\r\n"
			case "Quarterly":
				icalContent += "RRULE:FREQ=MONTHLY;INTERVAL=3\r\n"
			case "Annual":
				icalContent += "RRULE:FREQ=YEARLY;INTERVAL=1\r\n"
			}

			icalContent += "END:VEVENT\r\n"
		}
	}

	icalContent += "END:VCALENDAR\r\n"
	return icalContent
}
