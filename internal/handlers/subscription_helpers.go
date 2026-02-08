package handlers

import (
	"log/slog"
	"time"

	"subvault/internal/models"
	"subvault/internal/service"
)

// enrichWithCurrencyConversion adds currency conversion info to subscriptions
func (h *SubscriptionHandler) enrichWithCurrencyConversion(subscriptions []models.Subscription) []SubscriptionWithConversion {
	displayCurrency := h.preferences.GetCurrency()
	displaySymbol := h.preferences.GetCurrencySymbol()

	result := make([]SubscriptionWithConversion, len(subscriptions))

	for i := range subscriptions {
		// Create a copy of the subscription for modification; this pattern is correct for Go 1.22+
		sub := subscriptions[i]
		originalSymbol := service.CurrencySymbolForCode(sub.OriginalCurrency)
		enriched := SubscriptionWithConversion{
			Subscription:           &sub,
			DisplayCurrency:        displayCurrency,
			DisplayCurrencySymbol:  displaySymbol,
			OriginalCurrencySymbol: originalSymbol,
			ShowConversion:         false,
		}

		// Only show conversion if currencies differ
		if sub.OriginalCurrency != "" && sub.OriginalCurrency != displayCurrency {
			if convertedCost, err := h.currencyService.ConvertAmount(sub.Cost, sub.OriginalCurrency, displayCurrency); err == nil {
				enriched.ConvertedCost = convertedCost
				enriched.ConvertedAnnualCost = convertedCost * h.getScheduleMultiplier(sub.Schedule)
				enriched.ConvertedMonthlyCost = enriched.ConvertedAnnualCost / 12
				enriched.ShowConversion = true
			}
		} else {
			// Same currency or no conversion needed
			enriched.ConvertedCost = sub.Cost
			enriched.ConvertedAnnualCost = sub.AnnualCost()
			enriched.ConvertedMonthlyCost = sub.MonthlyCost()
		}

		result[i] = enriched
	}

	return result
}

// isHighCostWithCurrency checks if a subscription is high-cost, respecting currency conversion
// The threshold is in the user's display currency, so we convert the subscription's monthly cost
// to the display currency before comparing
func (h *SubscriptionHandler) isHighCostWithCurrency(subscription *models.Subscription) bool {
	threshold := h.settings.GetFloatSettingWithDefault("high_cost_threshold", 50.0)
	displayCurrency := h.preferences.GetCurrency()

	// Get monthly cost in subscription's original currency
	monthlyCost := subscription.MonthlyCost()

	// If currencies match, compare directly
	if subscription.OriginalCurrency == displayCurrency {
		return monthlyCost > threshold
	}

	// Convert monthly cost to display currency
	convertedMonthlyCost, err := h.currencyService.ConvertAmount(monthlyCost, subscription.OriginalCurrency, displayCurrency)
	if err != nil {
		// If conversion fails, fall back to direct comparison
		// Note: This may not be accurate if currencies differ, but prevents silent failures
		// The warning log helps identify when this fallback is used
		slog.Warn("failed to convert currency for high-cost check, using direct comparison", "from", subscription.OriginalCurrency, "to", displayCurrency, "error", err)
		return monthlyCost > threshold
	}

	// Compare converted monthly cost against threshold
	return convertedMonthlyCost > threshold
}

// fetchAndSetLogo fetches a logo for a subscription if URL is provided and icon_url is empty
// This is a helper method to avoid code duplication between create and update handlers
func (h *SubscriptionHandler) fetchAndSetLogo(subscription *models.Subscription) {
	if subscription.URL == "" || subscription.IconURL != "" {
		return
	}

	iconURL, err := h.logoService.FetchLogoFromURL(subscription.URL)
	if err == nil && iconURL != "" {
		subscription.IconURL = iconURL
		slog.Info("fetched logo", "url", subscription.URL, "iconURL", iconURL)
	} else if err != nil {
		slog.Error("failed to fetch logo", "url", subscription.URL, "error", err)
	}
}

// getScheduleMultiplier returns the annual multiplier for a schedule
func (h *SubscriptionHandler) getScheduleMultiplier(schedule string) float64 {
	switch schedule {
	case "Annual":
		return 1
	case "Quarterly":
		return 4
	case "Monthly":
		return 12
	case "Weekly":
		return 52
	case "Daily":
		return 365
	default:
		return 12
	}
}

// checkBudgetExceeded checks if the monthly budget has been exceeded and sends alerts
func (h *SubscriptionHandler) checkBudgetExceeded() {
	budget := h.settings.GetFloatSettingWithDefault("monthly_budget", 0)
	if budget <= 0 {
		return
	}

	stats, err := h.service.GetStats()
	if err != nil {
		return
	}

	if stats.TotalMonthlySpend > budget {
		currencySymbol := h.preferences.GetCurrencySymbol()
		if h.emailService != nil {
			go h.emailService.SendBudgetExceededAlert(stats.TotalMonthlySpend, budget, currencySymbol)
		}
		if h.shoutrrrService != nil {
			go h.shoutrrrService.SendBudgetExceededAlert(stats.TotalMonthlySpend, budget, currencySymbol)
		}
	}
}

// parseDatePtr parses a date string in "2006-01-02" format and returns a pointer to time.Time.
// Returns nil if the string is empty or if parsing fails.
// Logs parsing errors for debugging purposes.
func parseDatePtr(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	if date, err := time.Parse("2006-01-02", dateStr); err == nil {
		return &date
	}
	// Log parsing errors for debugging (invalid date format from form)
	slog.Warn("failed to parse date string", "dateStr", dateStr, "expectedFormat", "YYYY-MM-DD")
	return nil
}

// Helper function to format date pointers
func formatDate(date *time.Time) string {
	if date == nil {
		return ""
	}
	return date.Format("2006-01-02")
}
