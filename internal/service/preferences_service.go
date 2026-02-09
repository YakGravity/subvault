package service

import (
	"fmt"
)

type PreferencesService struct {
	settings     *SettingsService
	langProvider LanguageProvider
}

func NewPreferencesService(settings *SettingsService, langProvider LanguageProvider) *PreferencesService {
	return &PreferencesService{settings: settings, langProvider: langProvider}
}

// GetTheme retrieves the current theme setting
func (p *PreferencesService) GetTheme() (string, error) {
	theme, ok := p.settings.GetCached(SettingKeyTheme)
	if !ok || theme == "" {
		return "default", nil
	}
	return theme, nil
}

// SetTheme saves the theme preference
func (p *PreferencesService) SetTheme(theme string) error {
	defer p.settings.InvalidateCache()
	return p.settings.Repo().Set(SettingKeyTheme, theme)
}

// IsDarkModeEnabled returns whether dark mode is enabled
func (p *PreferencesService) IsDarkModeEnabled() bool {
	return p.settings.GetBoolSettingWithDefault(SettingKeyDarkMode, false)
}

// SetDarkMode saves the dark mode preference
func (p *PreferencesService) SetDarkMode(enabled bool) error {
	return p.settings.SetBoolSetting(SettingKeyDarkMode, enabled)
}

// SetCurrency saves the currency preference
func (p *PreferencesService) SetCurrency(currency string) error {
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
	defer p.settings.InvalidateCache()
	return p.settings.Repo().Set(SettingKeyCurrency, currency)
}

// GetCurrency retrieves the currency preference
func (p *PreferencesService) GetCurrency() string {
	currency, ok := p.settings.GetCached(SettingKeyCurrency)
	if !ok || currency == "" {
		return "USD"
	}
	return currency
}

// GetCurrencySymbol returns the symbol for the current currency
func (p *PreferencesService) GetCurrencySymbol() string {
	return CurrencySymbolForCode(p.GetCurrency())
}

// SetLanguage saves the language preference
func (p *PreferencesService) SetLanguage(lang string) error {
	isValid := false
	for _, l := range p.langProvider.SupportedLanguages() {
		if lang == l {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid language: %s", lang)
	}
	defer p.settings.InvalidateCache()
	return p.settings.Repo().Set(SettingKeyLanguage, lang)
}

// GetLanguage retrieves the language preference
func (p *PreferencesService) GetLanguage() string {
	lang, ok := p.settings.GetCached(SettingKeyLanguage)
	if !ok || lang == "" {
		return "en"
	}
	return lang
}

// SetDateFormat saves the date format preference
func (p *PreferencesService) SetDateFormat(format string) error {
	p.settings.InvalidateCache()
	return p.settings.Repo().Set(SettingKeyDateFormat, format)
}

// GetDateFormat retrieves the date format preference (Go format string).
// Returns empty string if not set (locale default will be used).
func (p *PreferencesService) GetDateFormat() string {
	val, ok := p.settings.GetCached(SettingKeyDateFormat)
	if !ok || val == "" {
		return ""
	}
	return val
}
