package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
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
	SettingKeyAuthResetToken    = "auth_reset_token"
	SettingKeyAuthResetExpiry   = "auth_reset_token_expiry"
	SettingKeyShoutrrrConfig    = "shoutrrr_config"
	SettingKeyPushoverConfig    = "pushover_config"
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

// getCached returns a cached setting value.
// Returns ("", false) if key is not found.
func (s *SettingsService) getCached(key string) (string, bool) {
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

// invalidateCache clears the settings cache
func (s *SettingsService) invalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastLoad = time.Time{}
}

// SaveSMTPConfig saves SMTP configuration
func (s *SettingsService) SaveSMTPConfig(config *models.SMTPConfig) error {
	// Convert to JSON
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	defer s.invalidateCache()
	return s.repo.Set(SettingKeySMTPConfig, string(data))
}

// GetSMTPConfig retrieves SMTP configuration
func (s *SettingsService) GetSMTPConfig() (*models.SMTPConfig, error) {
	data, ok := s.getCached(SettingKeySMTPConfig)
	if !ok {
		return nil, fmt.Errorf("smtp_config not found")
	}

	var config models.SMTPConfig
	err := json.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// SetBoolSetting saves a boolean setting
func (s *SettingsService) SetBoolSetting(key string, value bool) error {
	defer s.invalidateCache()
	return s.repo.Set(key, fmt.Sprintf("%t", value))
}

// GetBoolSetting retrieves a boolean setting
func (s *SettingsService) GetBoolSetting(key string, defaultValue bool) (bool, error) {
	value, ok := s.getCached(key)
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
	defer s.invalidateCache()
	return s.repo.Set(key, strconv.Itoa(value))
}

// GetIntSetting retrieves an integer setting
func (s *SettingsService) GetIntSetting(key string, defaultValue int) (int, error) {
	value, ok := s.getCached(key)
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
	defer s.invalidateCache()
	return s.repo.Set(key, fmt.Sprintf("%.2f", value))
}

// GetFloatSetting retrieves a float setting
func (s *SettingsService) GetFloatSetting(key string, defaultValue float64) (float64, error) {
	value, ok := s.getCached(key)
	if !ok {
		return defaultValue, nil
	}
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue, err
	}
	return floatValue, nil
}

// GetTheme retrieves the current theme setting
func (s *SettingsService) GetTheme() (string, error) {
	theme, ok := s.getCached(SettingKeyTheme)
	if !ok || theme == "" {
		return "default", nil
	}
	return theme, nil
}

// SetTheme saves the theme preference
func (s *SettingsService) SetTheme(theme string) error {
	defer s.invalidateCache()
	return s.repo.Set(SettingKeyTheme, theme)
}

// GetFloatSettingWithDefault retrieves a float setting with default
func (s *SettingsService) GetFloatSettingWithDefault(key string, defaultValue float64) float64 {
	value, err := s.GetFloatSetting(key, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

// CreateAPIKey creates a new API key
func (s *SettingsService) CreateAPIKey(name, key string) (*models.APIKey, error) {
	apiKey := &models.APIKey{
		Name: name,
		Key:  key,
	}
	return s.repo.CreateAPIKey(apiKey)
}

// GetAllAPIKeys retrieves all API keys
func (s *SettingsService) GetAllAPIKeys() ([]models.APIKey, error) {
	return s.repo.GetAllAPIKeys()
}

// DeleteAPIKey deletes an API key
func (s *SettingsService) DeleteAPIKey(id uint) error {
	return s.repo.DeleteAPIKey(id)
}

// ValidateAPIKey checks if an API key is valid and updates usage
func (s *SettingsService) ValidateAPIKey(key string) (*models.APIKey, error) {
	apiKey, err := s.repo.GetAPIKeyByKey(key)
	if err != nil {
		return nil, err
	}

	// Update usage stats
	err = s.repo.UpdateAPIKeyUsage(apiKey.ID)
	if err != nil {
		return nil, err
	}

	return apiKey, nil
}

// SetCurrency saves the currency preference
func (s *SettingsService) SetCurrency(currency string) error {
	// Validate currency using shared constant
	isValid := false
	for _, c := range SupportedCurrencies {
		if currency == c {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid currency: %s", currency)
	}
	defer s.invalidateCache()
	return s.repo.Set(SettingKeyCurrency, currency)
}

// GetCurrency retrieves the currency preference
func (s *SettingsService) GetCurrency() string {
	currency, ok := s.getCached(SettingKeyCurrency)
	if !ok || currency == "" {
		return "USD"
	}
	return currency
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

// GetCurrencySymbol returns the symbol for the current currency
func (s *SettingsService) GetCurrencySymbol() string {
	return CurrencySymbolForCode(s.GetCurrency())
}

// SetDarkMode saves the dark mode preference
func (s *SettingsService) SetDarkMode(enabled bool) error {
	return s.SetBoolSetting(SettingKeyDarkMode, enabled)
}

// IsDarkModeEnabled returns whether dark mode is enabled
func (s *SettingsService) IsDarkModeEnabled() bool {
	return s.GetBoolSettingWithDefault(SettingKeyDarkMode, false)
}

// Auth-related methods

// IsAuthEnabled returns whether authentication is enabled
func (s *SettingsService) IsAuthEnabled() bool {
	return s.GetBoolSettingWithDefault(SettingKeyAuthEnabled, false)
}

// SetAuthEnabled enables or disables authentication
func (s *SettingsService) SetAuthEnabled(enabled bool) error {
	return s.SetBoolSetting(SettingKeyAuthEnabled, enabled)
}

// GetAuthUsername returns the configured admin username
func (s *SettingsService) GetAuthUsername() (string, error) {
	val, ok := s.getCached(SettingKeyAuthUsername)
	if !ok {
		return "", fmt.Errorf("auth_username not found")
	}
	return val, nil
}

// SetAuthUsername sets the admin username
func (s *SettingsService) SetAuthUsername(username string) error {
	defer s.invalidateCache()
	return s.repo.Set(SettingKeyAuthUsername, username)
}

// HashPassword hashes a password using bcrypt
func (s *SettingsService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// SetAuthPassword hashes and stores the admin password
func (s *SettingsService) SetAuthPassword(password string) error {
	hash, err := s.HashPassword(password)
	if err != nil {
		return err
	}
	defer s.invalidateCache()
	return s.repo.Set(SettingKeyAuthPasswordHash, hash)
}

// ValidatePassword checks if a password matches the stored hash
func (s *SettingsService) ValidatePassword(password string) error {
	hash, ok := s.getCached(SettingKeyAuthPasswordHash)
	if !ok {
		return fmt.Errorf("no password configured")
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GetOrGenerateSessionSecret returns the session secret, generating one if it doesn't exist
func (s *SettingsService) GetOrGenerateSessionSecret() (string, error) {
	secret, ok := s.getCached(SettingKeyAuthSessionSecret)
	if ok && secret != "" {
		return secret, nil
	}

	// Generate a new 64-byte random secret
	bytes := make([]byte, 64)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	secret = base64.URLEncoding.EncodeToString(bytes)

	// Save it
	if err := s.repo.Set(SettingKeyAuthSessionSecret, secret); err != nil {
		return "", err
	}
	s.invalidateCache()

	return secret, nil
}

// SetupAuth sets up authentication with username and password
func (s *SettingsService) SetupAuth(username, password string) error {
	// Set username
	if err := s.SetAuthUsername(username); err != nil {
		return err
	}

	// Set password
	if err := s.SetAuthPassword(password); err != nil {
		return err
	}

	// Generate session secret
	if _, err := s.GetOrGenerateSessionSecret(); err != nil {
		return err
	}

	// Enable auth
	return s.SetAuthEnabled(true)
}

// DisableAuth disables authentication and removes credentials
func (s *SettingsService) DisableAuth() error {
	// Disable auth first
	if err := s.SetAuthEnabled(false); err != nil {
		return err
	}

	// Optionally clear credentials (commented out to allow re-enabling without re-entering)
	// s.repo.Delete(SettingKeyAuthUsername)
	// s.repo.Delete(SettingKeyAuthPasswordHash)

	return nil
}

// GenerateResetToken generates a password reset token
func (s *SettingsService) GenerateResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(bytes)

	if err := s.repo.Set(SettingKeyAuthResetToken, token); err != nil {
		return "", err
	}

	expiry := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	if err := s.repo.Set(SettingKeyAuthResetExpiry, expiry); err != nil {
		return "", err
	}

	s.invalidateCache()
	return token, nil
}

// ValidateResetToken checks if a reset token is valid
func (s *SettingsService) ValidateResetToken(token string) error {
	storedToken, ok := s.getCached(SettingKeyAuthResetToken)
	if !ok || subtle.ConstantTimeCompare([]byte(storedToken), []byte(token)) != 1 {
		return fmt.Errorf("invalid token")
	}

	expiryStr, ok := s.getCached(SettingKeyAuthResetExpiry)
	if !ok {
		return fmt.Errorf("token expired")
	}

	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil || time.Now().After(expiry) {
		return fmt.Errorf("token expired")
	}

	return nil
}

// ClearResetToken removes the reset token after use
func (s *SettingsService) ClearResetToken() error {
	s.repo.Delete(SettingKeyAuthResetToken)
	s.repo.Delete(SettingKeyAuthResetExpiry)
	s.invalidateCache()
	return nil
}

// SaveShoutrrrConfig saves Shoutrrr configuration
func (s *SettingsService) SaveShoutrrrConfig(config *models.ShoutrrrConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	defer s.invalidateCache()
	return s.repo.Set(SettingKeyShoutrrrConfig, string(data))
}

// GetShoutrrrConfig retrieves Shoutrrr configuration
func (s *SettingsService) GetShoutrrrConfig() (*models.ShoutrrrConfig, error) {
	data, ok := s.getCached(SettingKeyShoutrrrConfig)
	if !ok {
		return nil, fmt.Errorf("shoutrrr_config not found")
	}

	var config models.ShoutrrrConfig
	err := json.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// MigratePushoverToShoutrrr migrates existing Pushover config to Shoutrrr format
func (s *SettingsService) MigratePushoverToShoutrrr() error {
	data, ok := s.getCached(SettingKeyPushoverConfig)
	if !ok {
		return nil // No Pushover config exists, nothing to migrate
	}

	var oldConfig struct {
		UserKey  string `json:"pushover_user_key"`
		AppToken string `json:"pushover_app_token"`
	}
	if err := json.Unmarshal([]byte(data), &oldConfig); err != nil {
		return nil // Invalid config, skip migration
	}

	if oldConfig.UserKey == "" || oldConfig.AppToken == "" {
		return nil // Empty config, skip migration
	}

	// Check if Shoutrrr config already exists
	if existing, err := s.GetShoutrrrConfig(); err == nil && len(existing.URLs) > 0 {
		return nil // Already migrated
	}

	// Convert to Shoutrrr Pushover URL format
	shoutrrrURL := fmt.Sprintf("pushover://shoutrrr:%s@%s/", oldConfig.AppToken, oldConfig.UserKey)

	newConfig := &models.ShoutrrrConfig{
		URLs: []string{shoutrrrURL},
	}

	if err := s.SaveShoutrrrConfig(newConfig); err != nil {
		return fmt.Errorf("failed to save migrated Shoutrrr config: %w", err)
	}

	// Delete old Pushover config
	s.repo.Delete(SettingKeyPushoverConfig)
	slog.Info("migrated Pushover config to Shoutrrr URL format")

	return nil
}

// SupportedLanguages defines the available UI languages
var SupportedLanguages = []string{"en", "de"}

// SetLanguage saves the language preference
func (s *SettingsService) SetLanguage(lang string) error {
	isValid := false
	for _, l := range SupportedLanguages {
		if lang == l {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid language: %s", lang)
	}
	defer s.invalidateCache()
	return s.repo.Set(SettingKeyLanguage, lang)
}

// GetLanguage retrieves the language preference
func (s *SettingsService) GetLanguage() string {
	lang, ok := s.getCached(SettingKeyLanguage)
	if !ok || lang == "" {
		return "en"
	}
	return lang
}

// SetDateFormat saves the date format preference
func (s *SettingsService) SetDateFormat(format string) error {
	s.invalidateCache()
	return s.repo.Set(SettingKeyDateFormat, format)
}

// GetDateFormat retrieves the date format preference (Go format string).
// Returns empty string if not set (locale default will be used).
func (s *SettingsService) GetDateFormat() string {
	val, ok := s.getCached(SettingKeyDateFormat)
	if !ok || val == "" {
		return ""
	}
	return val
}

// GenerateCalendarToken creates a new calendar feed token
func (s *SettingsService) GenerateCalendarToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := fmt.Sprintf("%x", bytes)
	if err := s.repo.Set(SettingKeyCalendarToken, token); err != nil {
		return "", err
	}
	s.invalidateCache()
	return token, nil
}

// GetCalendarToken retrieves the calendar feed token
func (s *SettingsService) GetCalendarToken() (string, error) {
	val, ok := s.getCached(SettingKeyCalendarToken)
	if !ok {
		return "", fmt.Errorf("calendar_token not found")
	}
	return val, nil
}

// RevokeCalendarToken deletes the calendar feed token
func (s *SettingsService) RevokeCalendarToken() error {
	defer s.invalidateCache()
	return s.repo.Set(SettingKeyCalendarToken, "")
}
