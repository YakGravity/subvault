package handlers

import (
	"log"
	"net/http"
	"strconv"
	"subtrackr/internal/models"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateSubscriptionRequest is the DTO for creating a subscription via API.
// Required fields are enforced via binding tags.
type CreateSubscriptionRequest struct {
	Name             string     `json:"name" binding:"required"`
	Cost             float64    `json:"cost" binding:"required,gt=0"`
	Schedule         string     `json:"schedule" binding:"required,oneof=Monthly Annual Weekly Daily Quarterly"`
	Status           string     `json:"status" binding:"required,oneof=Active Cancelled Paused Trial"`
	OriginalCurrency string     `json:"original_currency"`
	CategoryID       uint       `json:"category_id"`
	PaymentMethod    string     `json:"payment_method"`
	LoginName        string     `json:"login_name"`
	TaxRate          float64    `json:"tax_rate"`
	PriceType        string     `json:"price_type"`
	CustomerNumber   string     `json:"customer_number"`
	ContractNumber   string     `json:"contract_number"`
	StartDate        *time.Time `json:"start_date"`
	RenewalDate      *time.Time `json:"renewal_date"`
	CancellationDate *time.Time `json:"cancellation_date"`
	URL              string     `json:"url"`
	IconURL          string     `json:"icon_url"`
	Notes            string     `json:"notes"`
	Usage            string     `json:"usage" binding:"omitempty,oneof=High Medium Low None"`
}

// UpdateSubscriptionRequest is the DTO for partial updates via API.
// All fields are pointers so we can distinguish between "not provided" (nil) and "set to zero value".
type UpdateSubscriptionRequest struct {
	Name             *string    `json:"name"`
	Cost             *float64   `json:"cost" binding:"omitempty,gt=0"`
	Schedule         *string    `json:"schedule" binding:"omitempty,oneof=Monthly Annual Weekly Daily Quarterly"`
	Status           *string    `json:"status" binding:"omitempty,oneof=Active Cancelled Paused Trial"`
	OriginalCurrency *string    `json:"original_currency"`
	CategoryID       *uint      `json:"category_id"`
	PaymentMethod    *string    `json:"payment_method"`
	LoginName        *string    `json:"login_name"`
	TaxRate          *float64   `json:"tax_rate"`
	PriceType        *string    `json:"price_type"`
	CustomerNumber   *string    `json:"customer_number"`
	ContractNumber   *string    `json:"contract_number"`
	StartDate        *time.Time `json:"start_date"`
	RenewalDate      *time.Time `json:"renewal_date"`
	CancellationDate *time.Time `json:"cancellation_date"`
	URL              *string    `json:"url"`
	IconURL          *string    `json:"icon_url"`
	Notes            *string    `json:"notes"`
	Usage            *string    `json:"usage" binding:"omitempty,oneof=High Medium Low None"`
}

// CreateSubscriptionAPI handles creating a new subscription via JSON API
func (h *SubscriptionHandler) CreateSubscriptionAPI(c *gin.Context) {
	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default price type to "gross" if not provided
	priceType := req.PriceType
	if priceType == "" {
		priceType = "gross"
	}

	subscription := models.Subscription{
		Name:             req.Name,
		Cost:             req.Cost,
		Schedule:         req.Schedule,
		Status:           req.Status,
		OriginalCurrency: req.OriginalCurrency,
		CategoryID:       req.CategoryID,
		PaymentMethod:    req.PaymentMethod,
		LoginName:        req.LoginName,
		TaxRate:          req.TaxRate,
		PriceType:        priceType,
		CustomerNumber:   req.CustomerNumber,
		ContractNumber:   req.ContractNumber,
		StartDate:        req.StartDate,
		RenewalDate:      req.RenewalDate,
		CancellationDate: req.CancellationDate,
		URL:              req.URL,
		IconURL:          req.IconURL,
		Notes:            req.Notes,
		Usage:            req.Usage,
	}

	if subscription.OriginalCurrency == "" {
		subscription.OriginalCurrency = "USD"
	}

	h.fetchAndSetLogo(&subscription)

	created, err := h.service.Create(&subscription)
	if err != nil {
		log.Printf("API: Failed to create subscription: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Send high-cost alert if applicable
	if h.isHighCostWithCurrency(created) {
		subscriptionWithCategory, err := h.service.GetByID(created.ID)
		if err == nil && subscriptionWithCategory != nil {
			if err := h.emailService.SendHighCostAlert(subscriptionWithCategory); err != nil {
				log.Printf("Failed to send high-cost alert email: %v", err)
			}
			if err := h.shoutrrrService.SendHighCostAlert(subscriptionWithCategory); err != nil {
				log.Printf("Failed to send high-cost alert Shoutrrr notification: %v", err)
			}
		}
	}

	c.JSON(http.StatusCreated, created)
}

// UpdateSubscriptionAPI handles partial updates to a subscription via JSON API
func (h *SubscriptionHandler) UpdateSubscriptionAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	original, err := h.service.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	var req UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wasHighCost := h.isHighCostWithCurrency(original)

	// Merge: only overwrite fields that were provided (non-nil)
	subscription := *original
	if req.Name != nil {
		subscription.Name = *req.Name
	}
	if req.Cost != nil {
		subscription.Cost = *req.Cost
	}
	if req.Schedule != nil {
		subscription.Schedule = *req.Schedule
	}
	if req.Status != nil {
		subscription.Status = *req.Status
	}
	if req.OriginalCurrency != nil {
		subscription.OriginalCurrency = *req.OriginalCurrency
	}
	if req.CategoryID != nil {
		subscription.CategoryID = *req.CategoryID
	}
	if req.PaymentMethod != nil {
		subscription.PaymentMethod = *req.PaymentMethod
	}
	if req.LoginName != nil {
		subscription.LoginName = *req.LoginName
	}
	if req.TaxRate != nil {
		subscription.TaxRate = *req.TaxRate
	}
	if req.PriceType != nil {
		subscription.PriceType = *req.PriceType
	}
	if req.CustomerNumber != nil {
		subscription.CustomerNumber = *req.CustomerNumber
	}
	if req.ContractNumber != nil {
		subscription.ContractNumber = *req.ContractNumber
	}
	if req.StartDate != nil {
		subscription.StartDate = req.StartDate
	}
	if req.RenewalDate != nil {
		subscription.RenewalDate = req.RenewalDate
	}
	if req.CancellationDate != nil {
		subscription.CancellationDate = req.CancellationDate
	}
	if req.URL != nil {
		subscription.URL = *req.URL
	}
	if req.IconURL != nil {
		subscription.IconURL = *req.IconURL
	}
	if req.Notes != nil {
		subscription.Notes = *req.Notes
	}
	if req.Usage != nil {
		subscription.Usage = *req.Usage
	}

	// Fetch logo if URL changed or new URL without icon
	urlChanged := req.URL != nil && original.URL != subscription.URL
	if urlChanged || (subscription.URL != "" && subscription.IconURL == "") {
		h.fetchAndSetLogo(&subscription)
	}

	updated, err := h.service.Update(uint(id), &subscription)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Send high-cost alert if subscription became high-cost
	if updated != nil && !wasHighCost && h.isHighCostWithCurrency(updated) {
		subscriptionWithCategory, err := h.service.GetByID(updated.ID)
		if err == nil && subscriptionWithCategory != nil {
			if err := h.emailService.SendHighCostAlert(subscriptionWithCategory); err != nil {
				log.Printf("Failed to send high-cost alert email: %v", err)
			}
			if err := h.shoutrrrService.SendHighCostAlert(subscriptionWithCategory); err != nil {
				log.Printf("Failed to send high-cost alert Shoutrrr notification: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteSubscriptionAPI handles deleting a subscription via JSON API
func (h *SubscriptionHandler) DeleteSubscriptionAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	err = h.service.Delete(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Subscription deleted",
		"id":      id,
	})
}
