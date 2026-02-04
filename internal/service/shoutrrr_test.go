package service

import (
	"os"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Shoutrrr Test Credentials Usage:
//
// For unit tests (default): Tests use invalid URLs and will fail sending (expected behavior)
//
// For integration tests: Set environment variable before running tests:
//
//	export SHOUTRRR_URL="pushover://shoutrrr:token@userkey/"
//
// Integration tests will automatically skip if the variable is not provided.

func setupShoutrrrTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	err = db.AutoMigrate(
		&models.Settings{},
		&models.Category{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestShoutrrrService_SendHighCostAlert_NoConfig(t *testing.T) {
	db := setupShoutrrrTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	shoutrrrService := NewShoutrrrService(settingsService)

	subscription := &models.Subscription{
		Name:     "Test Subscription",
		Cost:     100.00,
		Schedule: "Monthly",
		Status:   "Active",
		Category: models.Category{Name: "Test"},
	}

	err := shoutrrrService.SendHighCostAlert(subscription)
	assert.Error(t, err, "Should return error when Shoutrrr is not configured")
}

func TestShoutrrrService_SendHighCostAlert_EnabledButNoConfig(t *testing.T) {
	db := setupShoutrrrTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	shoutrrrService := NewShoutrrrService(settingsService)

	settingsService.SetBoolSetting("high_cost_alerts", true)
	settingsService.SetCurrency("USD")

	subscription := &models.Subscription{
		Name:     "Test Subscription",
		Cost:     100.00,
		Schedule: "Monthly",
		Status:   "Active",
		Category: models.Category{Name: "Test"},
	}

	err := shoutrrrService.SendHighCostAlert(subscription)
	assert.Error(t, err, "Should return error when Shoutrrr is not configured")
}

func TestShoutrrrService_SendRenewalReminder_NoConfig(t *testing.T) {
	db := setupShoutrrrTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	shoutrrrService := NewShoutrrrService(settingsService)

	subscription := &models.Subscription{
		Name:        "Test Subscription",
		Cost:        10.00,
		Schedule:    "Monthly",
		Status:      "Active",
		RenewalDate: timePtr(time.Now().AddDate(0, 0, 3)),
		Category:    models.Category{Name: "Test"},
	}

	err := shoutrrrService.SendRenewalReminder(subscription, 3)
	assert.Error(t, err, "Should return error when Shoutrrr is not configured")
}

func TestShoutrrrService_SendRenewalReminder_EnabledButNoConfig(t *testing.T) {
	db := setupShoutrrrTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	shoutrrrService := NewShoutrrrService(settingsService)

	settingsService.SetBoolSetting("renewal_reminders", true)
	settingsService.SetCurrency("USD")

	subscription := &models.Subscription{
		Name:        "Test Subscription",
		Cost:        10.00,
		Schedule:    "Monthly",
		Status:      "Active",
		RenewalDate: timePtr(time.Now().AddDate(0, 0, 3)),
		Category:    models.Category{Name: "Test"},
	}

	err := shoutrrrService.SendRenewalReminder(subscription, 3)
	assert.Error(t, err, "Should return error when Shoutrrr is not configured")
}

func TestShoutrrrService_SendCancellationReminder_NoConfig(t *testing.T) {
	db := setupShoutrrrTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	shoutrrrService := NewShoutrrrService(settingsService)

	subscription := &models.Subscription{
		Name:             "Test Subscription",
		Cost:             10.00,
		Schedule:         "Monthly",
		Status:           "Active",
		CancellationDate: timePtr(time.Now().AddDate(0, 0, 3)),
		Category:         models.Category{Name: "Test"},
	}

	err := shoutrrrService.SendCancellationReminder(subscription, 3)
	assert.Error(t, err, "Should return error when Shoutrrr is not configured")
}

func TestShoutrrrService_SendHighCostAlert_WithInvalidURL(t *testing.T) {
	db := setupShoutrrrTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	shoutrrrService := NewShoutrrrService(settingsService)

	config := &models.ShoutrrrConfig{
		URLs: []string{"pushover://shoutrrr:invalidtoken@invaliduser/"},
	}
	settingsService.SaveShoutrrrConfig(config)
	settingsService.SetBoolSetting("high_cost_alerts", true)
	settingsService.SetCurrency("USD")

	subscription := &models.Subscription{
		Name:        "Netflix",
		Cost:        15.99,
		Schedule:    "Monthly",
		Status:      "Active",
		RenewalDate: timePtr(time.Now().AddDate(0, 0, 30)),
		Category:    models.Category{Name: "Entertainment"},
		URL:         "https://netflix.com",
	}

	err := shoutrrrService.SendHighCostAlert(subscription)
	assert.Error(t, err, "Should return error when Shoutrrr URL credentials are invalid")
}

func TestShoutrrrService_MigratePushoverToShoutrrr(t *testing.T) {
	db := setupShoutrrrTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)

	// Save old Pushover config
	settingsRepo.Set("pushover_config", `{"pushover_user_key":"testuser123","pushover_app_token":"testtoken456"}`)

	// Run migration
	err := settingsService.MigratePushoverToShoutrrr()
	assert.NoError(t, err, "Migration should succeed")

	// Verify new config
	config, err := settingsService.GetShoutrrrConfig()
	assert.NoError(t, err, "Should get Shoutrrr config after migration")
	assert.Len(t, config.URLs, 1, "Should have one URL")
	assert.Equal(t, "pushover://shoutrrr:testtoken456@testuser123/", config.URLs[0])

	// Verify old config was deleted
	_, err = settingsRepo.Get("pushover_config")
	assert.Error(t, err, "Old Pushover config should be deleted")
}

func TestShoutrrrService_MigratePushoverToShoutrrr_NoPushoverConfig(t *testing.T) {
	db := setupShoutrrrTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)

	// No Pushover config exists
	err := settingsService.MigratePushoverToShoutrrr()
	assert.NoError(t, err, "Migration should succeed silently when no Pushover config exists")
}

func TestShoutrrrService_MigratePushoverToShoutrrr_AlreadyMigrated(t *testing.T) {
	db := setupShoutrrrTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)

	// Save old Pushover config
	settingsRepo.Set("pushover_config", `{"pushover_user_key":"testuser","pushover_app_token":"testtoken"}`)

	// Save existing Shoutrrr config
	settingsService.SaveShoutrrrConfig(&models.ShoutrrrConfig{
		URLs: []string{"slack://token@channel"},
	})

	// Run migration
	err := settingsService.MigratePushoverToShoutrrr()
	assert.NoError(t, err, "Migration should skip silently when Shoutrrr config already exists")

	// Verify existing config was not overwritten
	config, err := settingsService.GetShoutrrrConfig()
	assert.NoError(t, err)
	assert.Len(t, config.URLs, 1)
	assert.Equal(t, "slack://token@channel", config.URLs[0])
}

// Integration test - only runs with SHOUTRRR_URL env var
func TestShoutrrrService_SendHighCostAlert_Integration(t *testing.T) {
	shoutrrrURL := os.Getenv("SHOUTRRR_URL")
	if shoutrrrURL == "" {
		t.Skip("Skipping integration test: SHOUTRRR_URL environment variable not set")
	}

	db := setupShoutrrrTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	shoutrrrService := NewShoutrrrService(settingsService)

	config := &models.ShoutrrrConfig{
		URLs: []string{shoutrrrURL},
	}
	settingsService.SaveShoutrrrConfig(config)
	settingsService.SetBoolSetting("high_cost_alerts", true)
	settingsService.SetCurrency("USD")

	subscription := &models.Subscription{
		Name:        "Test High Cost Subscription",
		Cost:        100.00,
		Schedule:    "Monthly",
		Status:      "Active",
		RenewalDate: timePtr(time.Now().AddDate(0, 0, 30)),
		Category:    models.Category{Name: "Test"},
		URL:         "https://example.com",
	}

	err := shoutrrrService.SendHighCostAlert(subscription)
	assert.NoError(t, err, "Should successfully send high cost alert with valid Shoutrrr URL")
}
