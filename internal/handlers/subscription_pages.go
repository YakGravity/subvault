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

	"subvault/internal/models"

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
		"CurrencySymbol":   h.preferences.GetCurrencySymbol(),
		"DarkMode":         h.preferences.IsDarkModeEnabled(),
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
		"CurrencySymbol": h.preferences.GetCurrencySymbol(),
		"DarkMode":       h.preferences.IsDarkModeEnabled(),
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

	// Project recurring renewal dates for the viewed month
	type Event struct {
		Name    string  `json:"name"`
		Cost    float64 `json:"cost"`
		ID      uint    `json:"id"`
		IconURL string  `json:"icon_url"`
		Color   string  `json:"color"`
		Type    string  `json:"type"`
	}
	viewStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	viewEnd := viewStart.AddDate(0, 1, 0)

	eventsByDate := make(map[string][]Event)
	for _, sub := range subscriptions {
		// Renewal events for Active, Trial and Paused subscriptions
		if sub.RenewalDate != nil && (sub.Status == "Active" || sub.Status == "Trial" || sub.Status == "Paused") {
			color := "mediumseagreen"
			switch sub.Status {
			case "Trial":
				color = "dodgerblue"
			case "Paused":
				color = "gray"
			}
			name := sub.Name
			if sub.Status != "Active" {
				name = fmt.Sprintf("%s (%s)", sub.Name, sub.Status)
			}

			// Calculate projected renewal dates in the viewed month
			dates := projectRenewalDates(*sub.RenewalDate, sub.Schedule, viewStart, viewEnd)
			for _, d := range dates {
				dateKey := d.Format("2006-01-02")
				eventsByDate[dateKey] = append(eventsByDate[dateKey], Event{
					Name:    name,
					Cost:    sub.Cost,
					ID:      sub.ID,
					IconURL: sub.IconURL,
					Color:   color,
					Type:    "renewal",
				})
			}
		}
		// Cancellation date events (one-time, no projection)
		if sub.CancellationDate != nil {
			if !sub.CancellationDate.Before(viewStart) && sub.CancellationDate.Before(viewEnd) {
				dateKey := sub.CancellationDate.Format("2006-01-02")
				eventsByDate[dateKey] = append(eventsByDate[dateKey], Event{
					Name:    fmt.Sprintf("%s - Cancel By", sub.Name),
					Cost:    sub.Cost,
					ID:      sub.ID,
					IconURL: sub.IconURL,
					Color:   "tomato",
					Type:    "cancel",
				})
			}
		}
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
		"CurrencySymbol": h.preferences.GetCurrencySymbol(),
		"DarkMode":       h.preferences.IsDarkModeEnabled(),
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
		"CurrencySymbol":    h.preferences.GetCurrencySymbol(),
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

// projectRenewalDates calculates all renewal dates that fall within [viewStart, viewEnd)
// by stepping forward or backward from the base renewal date using the subscription schedule.
func projectRenewalDates(baseDate time.Time, schedule string, viewStart, viewEnd time.Time) []time.Time {
	var step func(t time.Time, n int) time.Time
	switch schedule {
	case "Daily":
		step = func(t time.Time, n int) time.Time { return t.AddDate(0, 0, n) }
	case "Weekly":
		step = func(t time.Time, n int) time.Time { return t.AddDate(0, 0, 7*n) }
	case "Monthly":
		step = func(t time.Time, n int) time.Time { return t.AddDate(0, n, 0) }
	case "Quarterly":
		step = func(t time.Time, n int) time.Time { return t.AddDate(0, 3*n, 0) }
	case "Annual":
		step = func(t time.Time, n int) time.Time { return t.AddDate(n, 0, 0) }
	default:
		// Unknown schedule: just check if baseDate falls in range
		if !baseDate.Before(viewStart) && baseDate.Before(viewEnd) {
			return []time.Time{baseDate}
		}
		return nil
	}

	var dates []time.Time

	// Step forward from baseDate
	for i := 0; ; i++ {
		d := step(baseDate, i)
		if !d.Before(viewEnd) {
			break
		}
		if !d.Before(viewStart) {
			dates = append(dates, d)
		}
		// Safety: don't generate more than 31 dates for daily schedules
		if len(dates) > 31 {
			break
		}
	}

	// Step backward from baseDate (skip i=0 already handled above)
	for i := 1; ; i++ {
		d := step(baseDate, -i)
		if d.Before(viewStart) {
			break
		}
		if d.Before(viewEnd) {
			dates = append(dates, d)
		}
		if i > 366 {
			break
		}
	}

	return dates
}
