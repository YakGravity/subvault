package service

import "subvault/internal/models"

// SubscriptionServiceInterface defines the contract for subscription operations.
type SubscriptionServiceInterface interface {
	Create(subscription *models.Subscription) (*models.Subscription, error)
	GetAll() ([]models.Subscription, error)
	GetAllPaginated(limit, offset int) ([]models.Subscription, int64, error)
	GetAllSorted(sortBy, order string) ([]models.Subscription, error)
	GetByID(id uint) (*models.Subscription, error)
	Update(id uint, subscription *models.Subscription) (*models.Subscription, error)
	Delete(id uint) error
	Count() int64
	GetStats() (*models.Stats, error)
	GetAllCategories() ([]models.Category, error)
	GetDefaultCategory() (*models.Category, error)
	GetSubscriptionsNeedingReminders() (map[*models.Subscription]int, error)
	GetSubscriptionsNeedingCancellationReminders() (map[*models.Subscription]int, error)
}

// SettingsServiceInterface defines the contract for base settings operations (cache + typed get/set).
type SettingsServiceInterface interface {
	SetBoolSetting(key string, value bool) error
	GetBoolSetting(key string, defaultValue bool) (bool, error)
	GetBoolSettingWithDefault(key string, defaultValue bool) bool
	SetIntSetting(key string, value int) error
	GetIntSetting(key string, defaultValue int) (int, error)
	GetIntSettingWithDefault(key string, defaultValue int) int
	SetFloatSetting(key string, value float64) error
	GetFloatSetting(key string, defaultValue float64) (float64, error)
	GetFloatSettingWithDefault(key string, defaultValue float64) float64
}

// AuthServiceInterface defines the contract for authentication operations.
type AuthServiceInterface interface {
	IsAuthEnabled() bool
	SetAuthEnabled(enabled bool) error
	GetAuthUsername() (string, error)
	SetAuthUsername(username string) error
	HashPassword(password string) (string, error)
	SetAuthPassword(password string) error
	ValidatePassword(password string) error
	GetOrGenerateSessionSecret() (string, error)
	GetOrGenerateCSRFSecret() ([]byte, error)
	SetupAuth(username, password string) error
	DisableAuth() error
	GenerateResetToken() (string, error)
	ValidateResetToken(token string) error
	ClearResetToken() error
}

// APIKeyServiceInterface defines the contract for API key operations.
type APIKeyServiceInterface interface {
	CreateAPIKey(name, key string) (*models.APIKey, error)
	GetAllAPIKeys() ([]models.APIKey, error)
	DeleteAPIKey(id uint) error
	ValidateAPIKey(key string) (*models.APIKey, error)
}

// PreferencesServiceInterface defines the contract for user preference operations.
type PreferencesServiceInterface interface {
	GetTheme() (string, error)
	SetTheme(theme string) error
	IsDarkModeEnabled() bool
	SetDarkMode(enabled bool) error
	SetCurrency(currency string) error
	GetCurrency() string
	GetCurrencySymbol() string
	SetLanguage(lang string) error
	GetLanguage() string
	SetDateFormat(format string) error
	GetDateFormat() string
}

// NotificationConfigServiceInterface defines the contract for notification configuration operations.
type NotificationConfigServiceInterface interface {
	SaveSMTPConfig(config *models.SMTPConfig) error
	GetSMTPConfig() (*models.SMTPConfig, error)
	SaveShoutrrrConfig(config *models.ShoutrrrConfig) error
	GetShoutrrrConfig() (*models.ShoutrrrConfig, error)
	MigratePushoverToShoutrrr() error
}

// CalendarServiceInterface defines the contract for calendar token operations.
type CalendarServiceInterface interface {
	GenerateCalendarToken() (string, error)
	GetCalendarToken() (string, error)
	RevokeCalendarToken() error
}

// CurrencyServiceInterface defines the contract for currency conversion operations.
type CurrencyServiceInterface interface {
	GetExchangeRate(fromCurrency, toCurrency string) (float64, error)
	ConvertAmount(amount float64, fromCurrency, toCurrency string) (float64, error)
	RefreshRates() error
	GetStatus() ExchangeRateStatus
}

// CategoryServiceInterface defines the contract for category operations.
type CategoryServiceInterface interface {
	Create(category *models.Category) (*models.Category, error)
	GetAll() ([]models.Category, error)
	GetAllPaginated(limit, offset int) ([]models.Category, int64, error)
	GetByID(id uint) (*models.Category, error)
	Update(id uint, category *models.Category) (*models.Category, error)
	Delete(id uint) error
	GetDefault() (*models.Category, error)
}

// EmailServiceInterface defines the contract for email notification operations.
type EmailServiceInterface interface {
	SendEmail(subject, body string) error
	SendHighCostAlert(subscription *models.Subscription) error
	SendRenewalReminder(subscription *models.Subscription, daysUntilRenewal int) error
	SendCancellationReminder(subscription *models.Subscription, daysUntilCancellation int) error
	SendBudgetExceededAlert(totalSpend, budget float64, currencySymbol string) error
}

// ShoutrrrServiceInterface defines the contract for Shoutrrr push notification operations.
type ShoutrrrServiceInterface interface {
	SendTestNotification(urls []string) error
	SendHighCostAlert(subscription *models.Subscription) error
	SendRenewalReminder(subscription *models.Subscription, daysUntilRenewal int) error
	SendCancellationReminder(subscription *models.Subscription, daysUntilCancellation int) error
	SendBudgetExceededAlert(totalSpend, budget float64, currencySymbol string) error
}

// LogoServiceInterface defines the contract for logo fetching and validation operations.
type LogoServiceInterface interface {
	FetchLogoFromURL(websiteURL string) (string, error)
	GetLogoURL(iconURL, websiteURL string) string
	ValidateLogoURL(logoURL string) bool
	FetchAndValidateLogo(websiteURL string) (string, error)
	ExtractDomain(websiteURL string) string
	DownloadLogo(logoURL string) ([]byte, error)
}

// RenewalServiceInterface defines the contract for subscription renewal date calculation.
type RenewalServiceInterface interface {
	InitializeRenewalDate(sub *models.Subscription)
	RecalculateIfNeeded(existing, updated *models.Subscription)
}

// Compile-time interface satisfaction checks.
var _ SubscriptionServiceInterface = (*SubscriptionService)(nil)
var _ SettingsServiceInterface = (*SettingsService)(nil)
var _ AuthServiceInterface = (*AuthService)(nil)
var _ APIKeyServiceInterface = (*APIKeyService)(nil)
var _ PreferencesServiceInterface = (*PreferencesService)(nil)
var _ NotificationConfigServiceInterface = (*NotificationConfigService)(nil)
var _ CalendarServiceInterface = (*CalendarService)(nil)
var _ CurrencyServiceInterface = (*CurrencyService)(nil)
var _ CategoryServiceInterface = (*CategoryService)(nil)
var _ EmailServiceInterface = (*EmailService)(nil)
var _ ShoutrrrServiceInterface = (*ShoutrrrService)(nil)
var _ LogoServiceInterface = (*LogoService)(nil)
var _ RenewalServiceInterface = (*RenewalService)(nil)
