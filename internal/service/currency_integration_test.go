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

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.ExchangeRate{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestCurrencyService_ConvertAmount_SameCurrency(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewExchangeRateRepository(db)
	service := NewCurrencyService(repo)

	result, err := service.ConvertAmount(100.0, "USD", "USD")

	assert.NoError(t, err)
	assert.Equal(t, 100.0, result)
}

func TestCurrencyService_ConvertAmount_WithCachedRate(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewExchangeRateRepository(db)
	service := NewCurrencyService(repo)

	// Create EUR-based cached rates (matches ECB format)
	err := repo.SaveRates([]models.ExchangeRate{
		{BaseCurrency: "EUR", Currency: "EUR", Rate: 1.0, Date: time.Now()},
		{BaseCurrency: "EUR", Currency: "USD", Rate: 1.1, Date: time.Now()},
	})
	assert.NoError(t, err)

	// USD->EUR cross-rate: EUR_rate / USD_rate = 1.0 / 1.1
	result, err := service.ConvertAmount(100.0, "USD", "EUR")

	assert.NoError(t, err)
	assert.InDelta(t, 90.909, result, 0.01)
}

func TestCurrencyService_ConvertAmount_NoECBRate(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewExchangeRateRepository(db)
	service := NewCurrencyService(repo)

	// RUB has no ECB rate, conversion should fail
	result, err := service.ConvertAmount(100.0, "RUB", "EUR")

	assert.Error(t, err)
	assert.Equal(t, 0.0, result)
	assert.Contains(t, err.Error(), "not provided by ECB")
}

func TestCurrencyService_ConvertAmount_InvalidAmount(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewExchangeRateRepository(db)
	service := NewCurrencyService(repo)

	// Pre-cache EUR-based rates
	repo.SaveRates([]models.ExchangeRate{
		{BaseCurrency: "EUR", Currency: "EUR", Rate: 1.0, Date: time.Now()},
		{BaseCurrency: "EUR", Currency: "USD", Rate: 1.1, Date: time.Now()},
	})

	// Cross-rate USD->EUR = 1.0/1.1 ≈ 0.9091
	tests := []struct {
		name     string
		amount   float64
		expected float64
	}{
		{"Negative amount", -100.0, -90.909},
		{"Zero amount", 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ConvertAmount(tt.amount, "USD", "EUR")
			assert.NoError(t, err)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestCurrencyService_SupportedCurrencies(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewExchangeRateRepository(db)
	service := NewCurrencyService(repo)

	// Test that same-currency conversion works for all supported currencies
	for _, currency := range SupportedCurrencies {
		t.Run(currency, func(t *testing.T) {
			result, err := service.ConvertAmount(100.0, currency, currency)
			assert.NoError(t, err)
			assert.Equal(t, 100.0, result)
		})
	}
}

func TestHasECBRate(t *testing.T) {
	tests := []struct {
		currency string
		expected bool
	}{
		{"EUR", true},
		{"USD", true},
		{"GBP", true},
		{"CHF", true},
		{"JPY", true},
		{"RUB", false},
		{"COP", false},
		{"BDT", false},
		{"XYZ", false},
	}

	for _, tt := range tests {
		t.Run(tt.currency, func(t *testing.T) {
			assert.Equal(t, tt.expected, HasECBRate(tt.currency))
		})
	}
}

func TestCurrencyService_BDTCurrency(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewExchangeRateRepository(db)
	service := NewCurrencyService(repo)

	t.Run("BDT same currency conversion", func(t *testing.T) {
		result, err := service.ConvertAmount(100.0, "BDT", "BDT")
		assert.NoError(t, err)
		assert.Equal(t, 100.0, result)
	})

	t.Run("BDT in SupportedCurrencies list", func(t *testing.T) {
		found := false
		for _, currency := range SupportedCurrencies {
			if currency == "BDT" {
				found = true
				break
			}
		}
		assert.True(t, found, "BDT should be in SupportedCurrencies list")
	})
}

func TestSettingsService_GetCurrencySymbol_BDT(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	err = db.AutoMigrate(&models.Settings{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	preferencesService := NewPreferencesService(settingsService)

	err = preferencesService.SetCurrency("BDT")
	assert.NoError(t, err)

	symbol := preferencesService.GetCurrencySymbol()
	assert.Equal(t, "৳", symbol)

	currency := preferencesService.GetCurrency()
	assert.Equal(t, "BDT", currency)
}

func TestSettingsService_SetCurrency_Validation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	err = db.AutoMigrate(&models.Settings{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := NewSettingsService(settingsRepo)
	preferencesService := NewPreferencesService(settingsService)

	tests := []struct {
		name           string
		currency       string
		shouldSucceed  bool
		expectedSymbol string
	}{
		{"Valid BDT currency", "BDT", true, "৳"},
		{"Invalid currency", "XYZ", false, ""},
		{"Valid USD currency", "USD", true, "$"},
		{"Valid EUR currency", "EUR", true, "€"},
		{"Valid AUD currency", "AUD", true, "A$"},
		{"Valid NOK currency", "NOK", true, "kr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := preferencesService.SetCurrency(tt.currency)
			if tt.shouldSucceed {
				assert.NoError(t, err)
				if tt.expectedSymbol != "" {
					symbol := preferencesService.GetCurrencySymbol()
					assert.Equal(t, tt.expectedSymbol, symbol)
				}
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid currency")
			}
		})
	}
}
