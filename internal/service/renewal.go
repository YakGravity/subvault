package service

import (
	"time"

	"subvault/internal/models"
)

type RenewalService struct{}

func NewRenewalService() *RenewalService {
	return &RenewalService{}
}

// InitializeRenewalDate sets the renewal date for new active subscriptions
func (r *RenewalService) InitializeRenewalDate(sub *models.Subscription) {
	if sub.Status == "Active" && sub.RenewalDate == nil {
		sub.CalculateNextRenewalDate()
	}
}

// RecalculateIfNeeded recalculates the renewal date when relevant fields change
func (r *RenewalService) RecalculateIfNeeded(existing, updated *models.Subscription) {
	if updated.Status != "Active" {
		return
	}

	// If schedule changed, recalculate
	if existing.Schedule != updated.Schedule {
		updated.CalculateNextRenewalDate()
		return
	}

	// If start date changed, recalculate
	if startDateChanged(existing.StartDate, updated.StartDate) {
		updated.CalculateNextRenewalDate()
		return
	}

	// If renewal date is nil, calculate it
	if updated.RenewalDate == nil {
		updated.CalculateNextRenewalDate()
		return
	}

	// If renewal date has passed, advance it
	now := time.Now()
	if updated.RenewalDate.Before(now) || updated.RenewalDate.Equal(now) {
		updated.CalculateNextRenewalDate()
	}
}

func startDateChanged(old, new *time.Time) bool {
	if old == nil && new != nil {
		return true
	}
	if old != nil && new == nil {
		return true
	}
	if old != nil && new != nil && !old.Equal(*new) {
		return true
	}
	return false
}
