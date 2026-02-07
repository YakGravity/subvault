package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"subtrackr/internal/models"

	"github.com/gin-gonic/gin"
)

// Dashboard renders the main dashboard page
func (h *SubscriptionHandler) Dashboard(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		slog.Error("failed to get subscription stats", "error", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "An internal error occurred"})
		return
	}

	// Use subscriptions from GetStats (already loaded, avoids duplicate DB query)
	enrichedSubs := h.enrichWithCurrencyConversion(stats.AllSubscriptions)

	// Build upcoming renewals (next 5 active subs by renewal date)
	now := time.Now()
	var upcoming []SubscriptionWithConversion
	for _, sub := range enrichedSubs {
		if sub.Status == "Active" && sub.RenewalDate != nil && sub.RenewalDate.After(now) {
			upcoming = append(upcoming, sub)
		}
	}
	sort.Slice(upcoming, func(i, j int) bool {
		return upcoming[i].RenewalDate.Before(*upcoming[j].RenewalDate)
	})
	if len(upcoming) > 5 {
		upcoming = upcoming[:5]
	}

	data := baseTemplateData(c)
	mergeTemplateData(data, gin.H{
		"Title":            "Dashboard",
		"CurrentPage":      "dashboard",
		"Stats":            stats,
		"Subscriptions":    enrichedSubs,
		"UpcomingRenewals": upcoming,
		"CurrencySymbol":   h.settingsService.GetCurrencySymbol(),
		"DarkMode":         h.settingsService.IsDarkModeEnabled(),
	})
	c.HTML(http.StatusOK, "dashboard.html", data)
}

// SubscriptionsList renders the subscriptions list page
func (h *SubscriptionHandler) SubscriptionsList(c *gin.Context) {
	// Get sort parameters from query string
	sortBy := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")

	// Get sorted subscriptions
	subscriptions, err := h.service.GetAllSorted(sortBy, order)
	if err != nil {
		slog.Error("failed to get sorted subscriptions", "error", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "An internal error occurred"})
		return
	}

	// Enrich with currency conversion
	enrichedSubs := h.enrichWithCurrencyConversion(subscriptions)

	data := baseTemplateData(c)
	mergeTemplateData(data, gin.H{
		"Title":          "Subscriptions",
		"CurrentPage":    "subscriptions",
		"Subscriptions":  enrichedSubs,
		"CurrencySymbol": h.settingsService.GetCurrencySymbol(),
		"DarkMode":       h.settingsService.IsDarkModeEnabled(),
		"SortBy":         sortBy,
		"Order":          order,
	})
	c.HTML(http.StatusOK, "subscriptions.html", data)
}

// Calendar renders the calendar page with subscription renewal dates
func (h *SubscriptionHandler) Calendar(c *gin.Context) {
	// Get all subscriptions with renewal dates
	subscriptions, err := h.service.GetAll()
	if err != nil {
		slog.Error("failed to get subscriptions for calendar", "error", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "An internal error occurred"})
		return
	}

	// Filter subscriptions with renewal dates and group by date
	// Create a simplified structure for JavaScript
	type Event struct {
		Name    string  `json:"name"`
		Cost    float64 `json:"cost"`
		ID      uint    `json:"id"`
		IconURL string  `json:"icon_url"`
	}
	eventsByDate := make(map[string][]Event)
	for _, sub := range subscriptions {
		if sub.RenewalDate != nil && sub.Status == "Active" {
			dateKey := sub.RenewalDate.Format("2006-01-02")
			eventsByDate[dateKey] = append(eventsByDate[dateKey], Event{
				Name:    sub.Name,
				Cost:    sub.Cost,
				ID:      sub.ID,
				IconURL: sub.IconURL,
			})
		}
	}

	// Get current month/year or from query params
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if y := c.Query("year"); y != "" {
		if yInt, err := strconv.Atoi(y); err == nil {
			year = yInt
		}
	}
	if m := c.Query("month"); m != "" {
		if mInt, err := strconv.Atoi(m); err == nil {
			month = mInt
		}
	}

	// Validate month range
	if month < 1 {
		month = 1
	}
	if month > 12 {
		month = 12
	}

	// Calculate previous and next month
	firstOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	prevMonth := firstOfMonth.AddDate(0, -1, 0)
	nextMonth := firstOfMonth.AddDate(0, 1, 0)

	// Serialize events to JSON for JavaScript
	eventsJSON, _ := json.Marshal(eventsByDate)

	// Prevent caching to ensure calendar updates when navigating months
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	data := baseTemplateData(c)
	mergeTemplateData(data, gin.H{
		"Title":          "Calendar",
		"CurrentPage":    "calendar",
		"Year":           year,
		"Month":          month,
		"MonthName":      fmt.Sprintf("%s %d", translateMonth(c, month), year),
		"EventsByDate":   template.JS(string(eventsJSON)),
		"FirstOfMonth":   firstOfMonth,
		"PrevMonth":      prevMonth,
		"NextMonth":      nextMonth,
		"CurrencySymbol": h.settingsService.GetCurrencySymbol(),
		"DarkMode":       h.settingsService.IsDarkModeEnabled(),
	})
	c.HTML(http.StatusOK, "calendar.html", data)
}

// GetSubscriptionForm returns the subscription form (for add/edit)
func (h *SubscriptionHandler) GetSubscriptionForm(c *gin.Context) {
	var subscription *models.Subscription
	isEdit := false

	// Check if this is an edit form
	if idStr := c.Param("id"); idStr != "" {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err == nil {
			sub, err := h.service.GetByID(uint(id))
			if err == nil {
				subscription = sub
				isEdit = true
			}
		}
	}

	categories, err := h.service.GetAllCategories()
	if err != nil {
		categories = []models.Category{}
	}

	var defaultCategoryID uint
	if defaultCat, err := h.service.GetDefaultCategory(); err == nil {
		defaultCategoryID = defaultCat.ID
	}

	data := baseTemplateData(c)
	mergeTemplateData(data, gin.H{
		"Subscription":      subscription,
		"IsEdit":            isEdit,
		"CurrencySymbol":    h.settingsService.GetCurrencySymbol(),
		"Categories":        categories,
		"DefaultCategoryID": defaultCategoryID,
	})
	c.HTML(http.StatusOK, "subscription-form.html", data)
}

// translateMonth returns the localized month name for a given month number (1-12).
func translateMonth(c *gin.Context, month int) string {
	monthKeys := []string{
		"month_january", "month_february", "month_march",
		"month_april", "month_may", "month_june",
		"month_july", "month_august", "month_september",
		"month_october", "month_november", "month_december",
	}
	fallbacks := []string{
		"January", "February", "March",
		"April", "May", "June",
		"July", "August", "September",
		"October", "November", "December",
	}
	if month < 1 || month > 12 {
		return "Unknown"
	}
	return tr(c, monthKeys[month-1], fallbacks[month-1])
}
