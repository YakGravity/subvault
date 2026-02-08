package service

import (
	"testing"
	"time"

	"subtrackr/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestRenewalService_InitializeRenewalDate(t *testing.T) {
	rs := NewRenewalService()

	t.Run("Active subscription without renewal date", func(t *testing.T) {
		sub := &models.Subscription{
			Name:     "Test",
			Cost:     9.99,
			Schedule: "Monthly",
			Status:   "Active",
		}
		rs.InitializeRenewalDate(sub)
		assert.NotNil(t, sub.RenewalDate)
		assert.True(t, sub.RenewalDate.After(time.Now()))
	})

	t.Run("Active subscription with existing renewal date", func(t *testing.T) {
		renewal := time.Now().AddDate(0, 1, 0)
		sub := &models.Subscription{
			Name:        "Test",
			Cost:        9.99,
			Schedule:    "Monthly",
			Status:      "Active",
			RenewalDate: &renewal,
		}
		rs.InitializeRenewalDate(sub)
		assert.Equal(t, renewal.Format("2006-01-02"), sub.RenewalDate.Format("2006-01-02"))
	})

	t.Run("Cancelled subscription", func(t *testing.T) {
		sub := &models.Subscription{
			Name:     "Test",
			Cost:     9.99,
			Schedule: "Monthly",
			Status:   "Cancelled",
		}
		rs.InitializeRenewalDate(sub)
		assert.Nil(t, sub.RenewalDate)
	})
}

func TestRenewalService_InitializeRenewalDate_WithStartDate(t *testing.T) {
	rs := NewRenewalService()

	tests := []struct {
		name      string
		schedule  string
		startDate time.Time
	}{
		{
			name:      "Monthly with past start date",
			schedule:  "Monthly",
			startDate: time.Now().AddDate(0, -2, -15),
		},
		{
			name:      "Annual with past start date",
			schedule:  "Annual",
			startDate: time.Now().AddDate(0, -6, 0),
		},
		{
			name:      "Weekly with past start date",
			schedule:  "Weekly",
			startDate: time.Now().AddDate(0, 0, -10),
		},
		{
			name:      "Future start date",
			schedule:  "Monthly",
			startDate: time.Now().AddDate(0, 0, 7),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &models.Subscription{
				Name:      "Test",
				Cost:      9.99,
				Schedule:  tt.schedule,
				Status:    "Active",
				StartDate: &tt.startDate,
			}

			rs.InitializeRenewalDate(sub)

			assert.NotNil(t, sub.RenewalDate)
			assert.True(t, sub.RenewalDate.After(time.Now()), "Renewal date should be in the future")
		})
	}
}

func TestRenewalService_RecalculateIfNeeded_ScheduleChange(t *testing.T) {
	rs := NewRenewalService()

	startDate := time.Now().AddDate(0, -3, 0)
	renewalDate := time.Now().AddDate(0, 1, 0)

	existing := &models.Subscription{
		Schedule:    "Monthly",
		Status:      "Active",
		StartDate:   &startDate,
		RenewalDate: &renewalDate,
	}

	updated := &models.Subscription{
		Schedule:    "Annual",
		Status:      "Active",
		StartDate:   &startDate,
		RenewalDate: &renewalDate,
	}

	rs.RecalculateIfNeeded(existing, updated)

	assert.NotNil(t, updated.RenewalDate)
	assert.True(t, updated.RenewalDate.After(time.Now()))
	assert.Equal(t, startDate.Month(), updated.RenewalDate.Month())
	assert.Equal(t, startDate.Day(), updated.RenewalDate.Day())
}

func TestRenewalService_RecalculateIfNeeded_NoChange(t *testing.T) {
	rs := NewRenewalService()

	originalRenewal := time.Now().AddDate(0, 1, 0)

	existing := &models.Subscription{
		Schedule:    "Monthly",
		Status:      "Active",
		RenewalDate: &originalRenewal,
	}

	updated := &models.Subscription{
		Schedule:    "Monthly",
		Status:      "Active",
		Cost:        19.99,
		RenewalDate: &originalRenewal,
	}

	rs.RecalculateIfNeeded(existing, updated)

	assert.Equal(t, originalRenewal.Format("2006-01-02"), updated.RenewalDate.Format("2006-01-02"))
}

func TestRenewalService_RecalculateIfNeeded_NilRenewalDate(t *testing.T) {
	rs := NewRenewalService()

	existing := &models.Subscription{
		Schedule: "Monthly",
		Status:   "Active",
	}

	updated := &models.Subscription{
		Schedule: "Monthly",
		Status:   "Active",
	}

	rs.RecalculateIfNeeded(existing, updated)

	assert.NotNil(t, updated.RenewalDate)
	assert.True(t, updated.RenewalDate.After(time.Now()))
}

func TestRenewalService_RecalculateIfNeeded_ExpiredDate(t *testing.T) {
	rs := NewRenewalService()

	expired := time.Now().AddDate(0, 0, -1)

	existing := &models.Subscription{
		Schedule:    "Monthly",
		Status:      "Active",
		RenewalDate: &expired,
	}

	updated := &models.Subscription{
		Schedule:    "Monthly",
		Status:      "Active",
		RenewalDate: &expired,
	}

	rs.RecalculateIfNeeded(existing, updated)

	assert.NotNil(t, updated.RenewalDate)
	assert.True(t, updated.RenewalDate.After(time.Now()))
}

func TestRenewalService_RecalculateIfNeeded_StartDateChange(t *testing.T) {
	rs := NewRenewalService()

	oldStart := time.Now().AddDate(0, -3, 0)
	newStart := time.Now().AddDate(0, -1, 0)
	renewal := time.Now().AddDate(0, 1, 0)

	existing := &models.Subscription{
		Schedule:    "Monthly",
		Status:      "Active",
		StartDate:   &oldStart,
		RenewalDate: &renewal,
	}

	updated := &models.Subscription{
		Schedule:    "Monthly",
		Status:      "Active",
		StartDate:   &newStart,
		RenewalDate: &renewal,
	}

	rs.RecalculateIfNeeded(existing, updated)

	assert.NotNil(t, updated.RenewalDate)
	assert.True(t, updated.RenewalDate.After(time.Now()))
}

func TestRenewalService_RecalculateIfNeeded_InactiveStatus(t *testing.T) {
	rs := NewRenewalService()

	existing := &models.Subscription{
		Schedule: "Monthly",
		Status:   "Active",
	}

	updated := &models.Subscription{
		Schedule: "Monthly",
		Status:   "Cancelled",
	}

	rs.RecalculateIfNeeded(existing, updated)

	assert.Nil(t, updated.RenewalDate)
}
