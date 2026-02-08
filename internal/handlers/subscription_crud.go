package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"subvault/internal/models"

	"github.com/gin-gonic/gin"
)

// GetSubscriptions returns subscriptions as HTML fragments
func (h *SubscriptionHandler) GetSubscriptions(c *gin.Context) {
	// Get sort parameters from query string
	sortBy := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")

	// Get sorted subscriptions
	subscriptions, err := h.service.GetAllSorted(sortBy, order)
	if err != nil {
		slog.Error("failed to get subscriptions", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Enrich with currency conversion
	enrichedSubs := h.enrichWithCurrencyConversion(subscriptions)

	data := baseTemplateData(c)
	mergeTemplateData(data, gin.H{
		"Subscriptions":  enrichedSubs,
		"CurrencySymbol": h.preferences.GetCurrencySymbol(),
		"SortBy":         sortBy,
		"Order":          order,
	})
	c.HTML(http.StatusOK, "subscription-list.html", data)
}

// GetSubscriptionsAPI returns subscriptions as JSON for API calls
func (h *SubscriptionHandler) GetSubscriptionsAPI(c *gin.Context) {
	subscriptions, err := h.service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, subscriptions)
}

// GetSubscription returns a single subscription
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidID})
		return
	}

	subscription, err := h.service.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": ErrSubscriptionNotFound})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// CreateSubscription handles creating a new subscription
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	var subscription models.Subscription

	// Parse form data
	subscription.Name = c.PostForm("name")
	// Parse category_id as uint
	if categoryIDStr := c.PostForm("category_id"); categoryIDStr != "" {
		if categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32); err == nil {
			subscription.CategoryID = uint(categoryID)
		}
	}
	subscription.Schedule = c.PostForm("schedule")
	subscription.Status = c.PostForm("status")
	subscription.OriginalCurrency = c.PostForm("original_currency")
	if subscription.OriginalCurrency == "" {
		subscription.OriginalCurrency = "USD" // Default to USD
	}
	subscription.PaymentMethod = c.PostForm("payment_method")
	subscription.LoginName = c.PostForm("login_name")
	subscription.CustomerNumber = c.PostForm("customer_number")
	subscription.ContractNumber = c.PostForm("contract_number")
	subscription.URL = c.PostForm("url")
	subscription.IconURL = c.PostForm("icon_url") // Allow manual icon URL override
	subscription.Notes = c.PostForm("notes")
	subscription.Usage = c.PostForm("usage")

	// Parse cost
	if costStr := c.PostForm("cost"); costStr != "" {
		if cost, err := strconv.ParseFloat(costStr, 64); err == nil {
			subscription.Cost = cost
		}
	}

	// Parse tax rate
	if taxRateStr := c.PostForm("tax_rate"); taxRateStr != "" {
		if taxRate, err := strconv.ParseFloat(taxRateStr, 64); err == nil {
			subscription.TaxRate = taxRate
		}
	}

	// Parse price type (default to "gross")
	priceType := c.PostForm("price_type")
	if priceType == "" {
		priceType = "gross"
	}
	subscription.PriceType = priceType

	// Parse dates using helper function
	subscription.StartDate = parseDatePtr(c.PostForm("start_date"))
	subscription.RenewalDate = parseDatePtr(c.PostForm("renewal_date"))
	subscription.CancellationDate = parseDatePtr(c.PostForm("cancellation_date"))

	// Parse per-subscription notification settings
	subscription.RenewalReminder = c.PostForm("renewal_reminder") == "on"
	if days, err := strconv.Atoi(c.PostForm("renewal_reminder_days")); err == nil && days > 0 {
		subscription.RenewalReminderDays = days
	} else {
		subscription.RenewalReminderDays = 3
	}
	subscription.CancellationReminder = c.PostForm("cancellation_reminder") == "on"
	if days, err := strconv.Atoi(c.PostForm("cancellation_reminder_days")); err == nil && days > 0 {
		subscription.CancellationReminderDays = days
	} else {
		subscription.CancellationReminderDays = 7
	}
	subscription.HighCostAlert = c.PostForm("high_cost_alert") == "on"

	// Fetch logo synchronously before creation if URL is provided and icon_url is empty
	h.fetchAndSetLogo(&subscription)

	// Create subscription
	created, err := h.service.Create(&subscription)
	if err != nil {
		// Log the error for debugging
		slog.Error("failed to create subscription", "error", err)
		slog.Debug("subscription data", "name", subscription.Name, "categoryID", subscription.CategoryID, "status", subscription.Status, "schedule", subscription.Schedule)

		if c.GetHeader("HX-Request") != "" {
			c.Header("HX-Retarget", "#form-errors")
			c.HTML(http.StatusBadRequest, "form-errors.html", gin.H{
				"Error": err.Error(),
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	// Send high-cost alert email and Shoutrrr notification if applicable (per-subscription setting)
	if created.HighCostAlert && h.isHighCostWithCurrency(created) {
		subscriptionWithCategory, err := h.service.GetByID(created.ID)
		if err == nil && subscriptionWithCategory != nil {
			if err := h.emailService.SendHighCostAlert(subscriptionWithCategory); err != nil {
				slog.Error("failed to send high-cost alert email", "error", err)
			}
			if err := h.shoutrrrService.SendHighCostAlert(subscriptionWithCategory); err != nil {
				slog.Error("failed to send high-cost alert shoutrrr notification", "error", err)
			}
		}
	}

	// Check budget after creating subscription
	h.checkBudgetExceeded()

	if c.GetHeader("HX-Request") != "" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusCreated)
	} else {
		c.JSON(http.StatusCreated, created)
	}
}

// UpdateSubscription handles updating an existing subscription
func (h *SubscriptionHandler) UpdateSubscription(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidID})
		return
	}

	var subscription models.Subscription

	// Parse form data (similar to CreateSubscription)
	subscription.Name = c.PostForm("name")
	// Parse category_id as uint
	if categoryIDStr := c.PostForm("category_id"); categoryIDStr != "" {
		if categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32); err == nil {
			subscription.CategoryID = uint(categoryID)
		}
	}
	subscription.Schedule = c.PostForm("schedule")
	subscription.Status = c.PostForm("status")
	subscription.OriginalCurrency = c.PostForm("original_currency")
	if subscription.OriginalCurrency == "" {
		subscription.OriginalCurrency = "USD" // Default to USD
	}
	subscription.PaymentMethod = c.PostForm("payment_method")
	subscription.LoginName = c.PostForm("login_name")
	subscription.CustomerNumber = c.PostForm("customer_number")
	subscription.ContractNumber = c.PostForm("contract_number")
	subscription.URL = c.PostForm("url")
	subscription.IconURL = c.PostForm("icon_url") // Allow manual icon URL override
	subscription.Notes = c.PostForm("notes")
	subscription.Usage = c.PostForm("usage")

	// Parse cost
	if costStr := c.PostForm("cost"); costStr != "" {
		if cost, err := strconv.ParseFloat(costStr, 64); err == nil {
			subscription.Cost = cost
		}
	}

	// Parse tax rate
	if taxRateStr := c.PostForm("tax_rate"); taxRateStr != "" {
		if taxRate, err := strconv.ParseFloat(taxRateStr, 64); err == nil {
			subscription.TaxRate = taxRate
		}
	}

	// Parse price type (default to "gross")
	priceType := c.PostForm("price_type")
	if priceType == "" {
		priceType = "gross"
	}
	subscription.PriceType = priceType

	// Parse dates using helper function
	// Always parse renewal date if provided; let service/model layer handle schedule change logic
	subscription.StartDate = parseDatePtr(c.PostForm("start_date"))
	subscription.RenewalDate = parseDatePtr(c.PostForm("renewal_date"))
	subscription.CancellationDate = parseDatePtr(c.PostForm("cancellation_date"))

	// Parse per-subscription notification settings
	subscription.RenewalReminder = c.PostForm("renewal_reminder") == "on"
	if days, err := strconv.Atoi(c.PostForm("renewal_reminder_days")); err == nil && days > 0 {
		subscription.RenewalReminderDays = days
	} else {
		subscription.RenewalReminderDays = 3
	}
	subscription.CancellationReminder = c.PostForm("cancellation_reminder") == "on"
	if days, err := strconv.Atoi(c.PostForm("cancellation_reminder_days")); err == nil && days > 0 {
		subscription.CancellationReminderDays = days
	} else {
		subscription.CancellationReminderDays = 7
	}
	subscription.HighCostAlert = c.PostForm("high_cost_alert") == "on"

	// Get the original subscription to check if it was high-cost before update
	original, _ := h.service.GetByID(uint(id))
	wasHighCost := original != nil && h.isHighCostWithCurrency(original)

	// Preserve existing IconURL if not explicitly set in form
	if subscription.IconURL == "" && original != nil {
		subscription.IconURL = original.IconURL
	}

	// Check if URL changed - if so, we should fetch a new logo
	urlChanged := original != nil && original.URL != subscription.URL
	if urlChanged || (subscription.URL != "" && subscription.IconURL == "") {
		h.fetchAndSetLogo(&subscription)
	}

	// Update subscription
	updated, err := h.service.Update(uint(id), &subscription)
	if err != nil {
		c.Header("HX-Retarget", "#form-errors")
		c.HTML(http.StatusBadRequest, "form-errors.html", gin.H{
			"Error": err.Error(),
		})
		return
	}

	// Send high-cost alert if subscription became high-cost (per-subscription setting)
	if updated != nil && updated.HighCostAlert && !wasHighCost && h.isHighCostWithCurrency(updated) {
		subscriptionWithCategory, err := h.service.GetByID(updated.ID)
		if err == nil && subscriptionWithCategory != nil {
			if err := h.emailService.SendHighCostAlert(subscriptionWithCategory); err != nil {
				slog.Error("failed to send high-cost alert email", "error", err)
			}
			if err := h.shoutrrrService.SendHighCostAlert(subscriptionWithCategory); err != nil {
				slog.Error("failed to send high-cost alert shoutrrr notification", "error", err)
			}
		}
	}

	// Check budget after updating subscription
	h.checkBudgetExceeded()

	// Return success response that triggers a page refresh
	c.Header("HX-Refresh", "true")
	c.Status(http.StatusOK)
}

// DeleteSubscription handles deleting a subscription
func (h *SubscriptionHandler) DeleteSubscription(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidID})
		return
	}

	err = h.service.Delete(uint(id))
	if err != nil {
		slog.Error("failed to delete subscription", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Return success response that triggers a page refresh
	c.Header("HX-Refresh", "true")
	c.Status(http.StatusOK)
}
