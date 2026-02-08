package service

import (
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRenewalReminderTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(
		&models.Subscription{},
		&models.Category{},
		&models.Settings{},
		&models.ExchangeRate{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestSubscriptionService_GetSubscriptionsNeedingReminders(t *testing.T) {
	db := setupRenewalReminderTestDB(t)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	categoryService := NewCategoryService(categoryRepo)
	exchangeRateRepo := repository.NewExchangeRateRepository(db)
	currencyService := NewCurrencyService(exchangeRateRepo)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	preferencesService := NewPreferencesService(settingsService)
	renewalService := NewRenewalService()
	subscriptionService := NewSubscriptionService(subscriptionRepo, categoryService, currencyService, preferencesService, settingsService, renewalService)

	now := time.Now()

	tests := []struct {
		name          string
		subscriptions []models.Subscription
		expectedCount int
		description   string
	}{
		{
			name: "Subscription renewing in 3 days with 7 day reminder",
			subscriptions: []models.Subscription{
				{
					Name:                "Test Subscription 1",
					Cost:                10.00,
					Schedule:            "Monthly",
					Status:              "Active",
					RenewalDate:         timePtr(now.AddDate(0, 0, 3)),
					RenewalReminder:     true,
					RenewalReminderDays: 7,
				},
			},
			expectedCount: 1,
			description:   "Should find subscription renewing within reminder window",
		},
		{
			name: "Subscription renewing in 10 days with 7 day reminder",
			subscriptions: []models.Subscription{
				{
					Name:                "Test Subscription 2",
					Cost:                10.00,
					Schedule:            "Monthly",
					Status:              "Active",
					RenewalDate:         timePtr(now.AddDate(0, 0, 10)),
					RenewalReminder:     true,
					RenewalReminderDays: 7,
				},
			},
			expectedCount: 0,
			description:   "Should not find subscription outside reminder window",
		},
		{
			name: "Subscription renewing today",
			subscriptions: []models.Subscription{
				{
					Name:                "Test Subscription 3",
					Cost:                10.00,
					Schedule:            "Monthly",
					Status:              "Active",
					RenewalDate:         timePtr(now.Add(12 * time.Hour)),
					RenewalReminder:     true,
					RenewalReminderDays: 7,
				},
			},
			expectedCount: 1,
			description:   "Should find subscription renewing today (within 24 hours)",
		},
		{
			name: "Multiple subscriptions mixed settings",
			subscriptions: []models.Subscription{
				{
					Name:                "Test Subscription 4",
					Cost:                10.00,
					Schedule:            "Monthly",
					Status:              "Active",
					RenewalDate:         timePtr(now.AddDate(0, 0, 2)),
					RenewalReminder:     true,
					RenewalReminderDays: 7,
				},
				{
					Name:                "Test Subscription 5",
					Cost:                20.00,
					Schedule:            "Monthly",
					Status:              "Active",
					RenewalDate:         timePtr(now.AddDate(0, 0, 5)),
					RenewalReminder:     true,
					RenewalReminderDays: 7,
				},
				{
					Name:                "Test Subscription 6",
					Cost:                30.00,
					Schedule:            "Monthly",
					Status:              "Active",
					RenewalDate:         timePtr(now.AddDate(0, 0, 3)),
					RenewalReminder:     false, // Reminder disabled
					RenewalReminderDays: 7,
				},
			},
			expectedCount: 2,
			description:   "Should find only subscriptions with reminder enabled and within window",
		},
		{
			name: "Cancelled subscription should be excluded",
			subscriptions: []models.Subscription{
				{
					Name:                "Test Subscription 7",
					Cost:                10.00,
					Schedule:            "Monthly",
					Status:              "Cancelled",
					RenewalDate:         timePtr(now.AddDate(0, 0, 3)),
					RenewalReminder:     true,
					RenewalReminderDays: 7,
				},
			},
			expectedCount: 0,
			description:   "Should exclude cancelled subscriptions",
		},
		{
			name: "Subscription without renewal date should be excluded",
			subscriptions: []models.Subscription{
				{
					Name:                "Test Subscription 8",
					Cost:                10.00,
					Schedule:            "Monthly",
					Status:              "Active",
					RenewalDate:         nil,
					RenewalReminder:     true,
					RenewalReminderDays: 7,
				},
			},
			expectedCount: 0,
			description:   "Should exclude subscriptions without renewal date",
		},
		{
			name: "Zero reminder days should return empty",
			subscriptions: []models.Subscription{
				{
					Name:                "Test Subscription 9",
					Cost:                10.00,
					Schedule:            "Monthly",
					Status:              "Active",
					RenewalDate:         timePtr(now.AddDate(0, 0, 3)),
					RenewalReminder:     true,
					RenewalReminderDays: 0,
				},
			},
			expectedCount: 0,
			description:   "Should return empty when reminder days is 0",
		},
		{
			name: "Past renewal date should be excluded",
			subscriptions: []models.Subscription{
				{
					Name:                "Test Subscription 10",
					Cost:                10.00,
					Schedule:            "Monthly",
					Status:              "Active",
					RenewalDate:         timePtr(now.AddDate(0, 0, -1)),
					RenewalReminder:     true,
					RenewalReminderDays: 7,
				},
			},
			expectedCount: 0,
			description:   "Should exclude subscriptions with past renewal dates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db.Exec("DELETE FROM subscriptions")

			for _, sub := range tt.subscriptions {
				err := db.Create(&sub).Error
				assert.NoError(t, err, "Failed to create test subscription")
			}

			result, err := subscriptionService.GetSubscriptionsNeedingReminders()
			assert.NoError(t, err, "GetSubscriptionsNeedingReminders should not return error")
			assert.Equal(t, tt.expectedCount, len(result), tt.description)

			for sub, daysUntil := range result {
				assert.GreaterOrEqual(t, daysUntil, 0, "Days until renewal should be non-negative")
				assert.LessOrEqual(t, daysUntil, sub.RenewalReminderDays, "Days until renewal should be within reminder window")
				assert.Equal(t, "Active", sub.Status, "Subscription should be active")
				assert.NotNil(t, sub.RenewalDate, "Subscription should have renewal date")
			}
		})
	}
}

func TestEmailService_SendRenewalReminder_NoSMTP(t *testing.T) {
	db := setupRenewalReminderTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	preferencesService := NewPreferencesService(settingsService)
	notifConfigService := NewNotificationConfigService(settingsService, settingsRepo)
	emailService := NewEmailService(preferencesService, notifConfigService)

	subscription := &models.Subscription{
		Name:            "Test Subscription",
		Cost:            10.00,
		Schedule:        "Monthly",
		Status:          "Active",
		RenewalDate:     timePtr(time.Now().AddDate(0, 0, 3)),
		RenewalReminder: true,
	}

	// Should return error because SMTP is not configured
	err := emailService.SendRenewalReminder(subscription, 3)
	assert.Error(t, err, "Should return error when SMTP is not configured")
}

func TestEmailService_SendRenewalReminder_WithSMTPConfig(t *testing.T) {
	db := setupRenewalReminderTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	preferencesService := NewPreferencesService(settingsService)
	notifConfigService := NewNotificationConfigService(settingsService, settingsRepo)
	emailService := NewEmailService(preferencesService, notifConfigService)

	// Configure SMTP (using invalid config - we're just testing the logic, not actual email sending)
	smtpConfig := &models.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "test@example.com",
		FromName: "Test",
		To:       "recipient@example.com",
	}
	notifConfigService.SaveSMTPConfig(smtpConfig)

	subscription := &models.Subscription{
		Name:            "Test Subscription",
		Cost:            10.00,
		Schedule:        "Monthly",
		Status:          "Active",
		RenewalDate:     timePtr(time.Now().AddDate(0, 0, 3)),
		RenewalReminder: true,
	}

	// This will fail because we don't have a real SMTP server, but it should attempt to send
	err := emailService.SendRenewalReminder(subscription, 3)
	assert.Error(t, err, "Should return error when SMTP connection fails (expected in test)")
	assert.NotContains(t, err.Error(), "disabled", "Error should not be about being disabled")
}

func TestSubscriptionService_GetSubscriptionsNeedingReminders_DaysCalculation(t *testing.T) {
	db := setupRenewalReminderTestDB(t)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	categoryService := NewCategoryService(categoryRepo)
	exchangeRateRepo := repository.NewExchangeRateRepository(db)
	currencyService := NewCurrencyService(exchangeRateRepo)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	preferencesService := NewPreferencesService(settingsService)
	renewalService := NewRenewalService()
	subscriptionService := NewSubscriptionService(subscriptionRepo, categoryService, currencyService, preferencesService, settingsService, renewalService)

	now := time.Now()

	// Create subscription renewing in exactly 5 days
	renewalDate := now.AddDate(0, 0, 5)
	sub := &models.Subscription{
		Name:                "Test Subscription",
		Cost:                10.00,
		Schedule:            "Monthly",
		Status:              "Active",
		RenewalDate:         &renewalDate,
		RenewalReminder:     true,
		RenewalReminderDays: 7,
	}
	err := db.Create(sub).Error
	assert.NoError(t, err)

	result, err := subscriptionService.GetSubscriptionsNeedingReminders()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result), "Should find one subscription")

	for foundSub, daysUntil := range result {
		assert.Equal(t, sub.ID, foundSub.ID, "Should be the same subscription")
		assert.InDelta(t, 5, daysUntil, 1, "Days until renewal should be approximately 5")
	}
}

func TestSubscriptionService_GetSubscriptionsNeedingReminders_BoundaryCases(t *testing.T) {
	db := setupRenewalReminderTestDB(t)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	categoryService := NewCategoryService(categoryRepo)
	exchangeRateRepo := repository.NewExchangeRateRepository(db)
	currencyService := NewCurrencyService(exchangeRateRepo)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	preferencesService := NewPreferencesService(settingsService)
	renewalService := NewRenewalService()
	subscriptionService := NewSubscriptionService(subscriptionRepo, categoryService, currencyService, preferencesService, settingsService, renewalService)

	now := time.Now()

	tests := []struct {
		name         string
		renewalDate  time.Time
		reminderDays int
		shouldFind   bool
		description  string
	}{
		{
			name:         "Exactly at reminder window boundary",
			renewalDate:  now.AddDate(0, 0, 7),
			reminderDays: 7,
			shouldFind:   true,
			description:  "Should find subscription renewing exactly at reminder window boundary",
		},
		{
			name:         "Just outside reminder window",
			renewalDate:  now.AddDate(0, 0, 8),
			reminderDays: 7,
			shouldFind:   false,
			description:  "Should not find subscription just outside reminder window",
		},
		{
			name:         "Renewing tomorrow",
			renewalDate:  now.AddDate(0, 0, 1),
			reminderDays: 7,
			shouldFind:   true,
			description:  "Should find subscription renewing tomorrow",
		},
		{
			name:         "Renewing in 1 hour (less than 1 day)",
			renewalDate:  now.Add(1 * time.Hour),
			reminderDays: 7,
			shouldFind:   true,
			description:  "Should find subscription renewing in less than 1 day (counts as 0 days)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db.Exec("DELETE FROM subscriptions")

			sub := &models.Subscription{
				Name:                "Test Subscription",
				Cost:                10.00,
				Schedule:            "Monthly",
				Status:              "Active",
				RenewalDate:         &tt.renewalDate,
				RenewalReminder:     true,
				RenewalReminderDays: tt.reminderDays,
			}
			err := db.Create(sub).Error
			assert.NoError(t, err)

			result, err := subscriptionService.GetSubscriptionsNeedingReminders()
			assert.NoError(t, err)

			if tt.shouldFind {
				assert.Equal(t, 1, len(result), tt.description)
			} else {
				assert.Equal(t, 0, len(result), tt.description)
			}
		})
	}
}

func TestSubscriptionService_GetSubscriptionsNeedingReminders_DuplicatePrevention(t *testing.T) {
	db := setupRenewalReminderTestDB(t)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	categoryService := NewCategoryService(categoryRepo)
	exchangeRateRepo := repository.NewExchangeRateRepository(db)
	currencyService := NewCurrencyService(exchangeRateRepo)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	preferencesService := NewPreferencesService(settingsService)
	renewalService := NewRenewalService()
	subscriptionService := NewSubscriptionService(subscriptionRepo, categoryService, currencyService, preferencesService, settingsService, renewalService)

	now := time.Now()
	renewalDate := now.AddDate(0, 0, 5)
	lastReminderDate := now.AddDate(0, 0, -1)

	// Create subscription with reminder already sent for this renewal date
	sub := &models.Subscription{
		Name:                    "Test Subscription",
		Cost:                    10.00,
		Schedule:                "Monthly",
		Status:                  "Active",
		RenewalDate:             &renewalDate,
		RenewalReminder:         true,
		RenewalReminderDays:     7,
		LastReminderSent:        &lastReminderDate,
		LastReminderRenewalDate: &renewalDate,
	}
	err := db.Create(sub).Error
	assert.NoError(t, err)

	result, err := subscriptionService.GetSubscriptionsNeedingReminders()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(result), "Should not find subscription that already has reminder sent for this renewal date")

	// Update to within window with different renewal date
	newRenewalDate := now.AddDate(0, 0, 3)
	sub.RenewalDate = &newRenewalDate
	err = db.Save(sub).Error
	assert.NoError(t, err)

	// Should find it now because renewal date changed
	result, err = subscriptionService.GetSubscriptionsNeedingReminders()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result), "Should find subscription when renewal date changes")
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
