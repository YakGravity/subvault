package service

import (
	"fmt"
	"log/slog"
	"strconv"
	"subvault/internal/repository"
	"sync"
	"time"
)

const settingsCacheTTL = 30 * time.Second

// Setting key constants
const (
	SettingKeySMTPConfig        = "smtp_config"
	SettingKeyTheme             = "theme"
	SettingKeyCurrency          = "currency"
	SettingKeyDarkMode          = "dark_mode"
	SettingKeyLanguage          = "language"
	SettingKeyDateFormat        = "date_format"
	SettingKeyCalendarToken     = "calendar_token"
	SettingKeyAuthEnabled       = "auth_enabled"
	SettingKeyAuthUsername      = "auth_username"
	SettingKeyAuthPasswordHash  = "auth_password_hash"
	SettingKeyAuthSessionSecret = "auth_session_secret"
	SettingKeyCSRFSecret        = "csrf_secret"
	SettingKeyAuthResetToken    = "auth_reset_token"
	SettingKeyAuthResetExpiry   = "auth_reset_token_expiry"
	SettingKeyShoutrrrConfig    = "shoutrrr_config"
	SettingKeyPushoverConfig       = "pushover_config"
	SettingKeyCurrencyRefreshHours = "currency_refresh_hours"
)

type SettingsService struct {
	repo     *repository.SettingsRepository
	mu       sync.RWMutex
	cache    map[string]string
	lastLoad time.Time
}

func NewSettingsService(repo *repository.SettingsRepository) *SettingsService {
	return &SettingsService{
		repo:  repo,
		cache: make(map[string]string),
	}
}

// Repo returns the underlying repository for use by dependent services.
func (s *SettingsService) Repo() *repository.SettingsRepository {
	return s.repo
}

// loadCache loads all settings into the in-memory cache
func (s *SettingsService) loadCache() {
	settings, err := s.repo.GetAll()
	if err != nil {
		slog.Warn("failed to load settings cache", "error", err)
		return
	}
	s.cache = make(map[string]string, len(settings))
	for _, setting := range settings {
		s.cache[setting.Key] = setting.Value
	}
	s.lastLoad = time.Now()
}

// GetCached returns a cached setting value.
// Returns ("", false) if key is not found.
func (s *SettingsService) GetCached(key string) (string, bool) {
	s.mu.RLock()
	if time.Since(s.lastLoad) < settingsCacheTTL && s.lastLoad != (time.Time{}) {
		val, ok := s.cache[key]
		s.mu.RUnlock()
		return val, ok
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	// Double-check after acquiring write lock
	if time.Since(s.lastLoad) < settingsCacheTTL && s.lastLoad != (time.Time{}) {
		val, ok := s.cache[key]
		return val, ok
	}
	s.loadCache()
	val, ok := s.cache[key]
	return val, ok
}

// InvalidateCache clears the settings cache
func (s *SettingsService) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastLoad = time.Time{}
}

// SetBoolSetting saves a boolean setting
func (s *SettingsService) SetBoolSetting(key string, value bool) error {
	defer s.InvalidateCache()
	return s.repo.Set(key, fmt.Sprintf("%t", value))
}

// GetBoolSetting retrieves a boolean setting
func (s *SettingsService) GetBoolSetting(key string, defaultValue bool) (bool, error) {
	value, ok := s.GetCached(key)
	if !ok {
		return defaultValue, nil
	}
	return value == "true", nil
}

// GetBoolSettingWithDefault retrieves a boolean setting with default
func (s *SettingsService) GetBoolSettingWithDefault(key string, defaultValue bool) bool {
	value, err := s.GetBoolSetting(key, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

// SetIntSetting saves an integer setting
func (s *SettingsService) SetIntSetting(key string, value int) error {
	defer s.InvalidateCache()
	return s.repo.Set(key, strconv.Itoa(value))
}

// GetIntSetting retrieves an integer setting
func (s *SettingsService) GetIntSetting(key string, defaultValue int) (int, error) {
	value, ok := s.GetCached(key)
	if !ok {
		return defaultValue, nil
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue, err
	}
	return intValue, nil
}

// GetIntSettingWithDefault retrieves an integer setting with default
func (s *SettingsService) GetIntSettingWithDefault(key string, defaultValue int) int {
	value, err := s.GetIntSetting(key, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

// SetFloatSetting saves a float setting
func (s *SettingsService) SetFloatSetting(key string, value float64) error {
	defer s.InvalidateCache()
	return s.repo.Set(key, fmt.Sprintf("%.2f", value))
}

// GetFloatSetting retrieves a float setting
func (s *SettingsService) GetFloatSetting(key string, defaultValue float64) (float64, error) {
	value, ok := s.GetCached(key)
	if !ok {
		return defaultValue, nil
	}
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue, err
	}
	return floatValue, nil
}

// GetFloatSettingWithDefault retrieves a float setting with default
func (s *SettingsService) GetFloatSettingWithDefault(key string, defaultValue float64) float64 {
	value, err := s.GetFloatSetting(key, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

// CurrencySymbolForCode returns the symbol for a given currency code
func CurrencySymbolForCode(code string) string {
	switch code {
	case "EUR":
		return "€"
	case "PLN":
		return "zł"
	case "GBP":
		return "£"
	case "RUB":
		return "₽"
	case "JPY":
		return "¥"
	case "SEK":
		return "kr"
	case "INR":
		return "₹"
	case "CHF":
		return "Fr."
	case "BRL":
		return "R$"
	case "BDT":
		return "৳"
	case "AUD":
		return "A$"
	case "CAD":
		return "C$"
	case "CNY":
		return "¥"
	case "CZK":
		return "Kč"
	case "DKK":
		return "kr"
	case "HKD":
		return "HK$"
	case "HUF":
		return "Ft"
	case "IDR":
		return "Rp"
	case "ILS":
		return "₪"
	case "ISK":
		return "kr"
	case "KRW":
		return "₩"
	case "MXN":
		return "MX$"
	case "MYR":
		return "RM"
	case "NOK":
		return "kr"
	case "NZD":
		return "NZ$"
	case "PHP":
		return "₱"
	case "RON":
		return "lei"
	case "SGD":
		return "S$"
	case "THB":
		return "฿"
	case "TRY":
		return "₺"
	case "ZAR":
		return "R"
	case "COP":
		return "COL$"
	default:
		return "$"
	}
}

// SupportedLanguages defines the available UI languages
var SupportedLanguages = []string{"en", "de"}
