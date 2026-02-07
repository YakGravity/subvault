package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localeFS embed.FS

// I18nService manages translation bundles and localizer creation
type I18nService struct {
	bundle         *i18n.Bundle
	defaultLang    string
	supportedLangs []string
}

// NewI18nService creates and initializes the i18n service
func NewI18nService() *I18nService {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load embedded locale files
	localeFiles := []string{
		"locales/active.en.json",
		"locales/active.de.json",
	}

	for _, file := range localeFiles {
		data, err := localeFS.ReadFile(file)
		if err != nil {
			slog.Warn("failed to read locale file", "file", file, "error", err)
			continue
		}
		if _, err := bundle.ParseMessageFileBytes(data, file); err != nil {
			slog.Warn("failed to parse locale file", "file", file, "error", err)
		}
	}

	return &I18nService{
		bundle:         bundle,
		defaultLang:    "en",
		supportedLangs: []string{"en", "de"},
	}
}

// NewLocalizer creates a localizer for the given language with English fallback
func (s *I18nService) NewLocalizer(lang string) *i18n.Localizer {
	if lang == "" {
		lang = s.defaultLang
	}
	return i18n.NewLocalizer(s.bundle, lang, s.defaultLang)
}

// T translates a simple message by ID
func (s *I18nService) T(localizer *i18n.Localizer, messageID string) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: messageID,
	})
	if err != nil {
		return messageID
	}
	return msg
}

// TData translates a message with template data
func (s *I18nService) TData(localizer *i18n.Localizer, messageID string, data map[string]interface{}) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
	})
	if err != nil {
		return messageID
	}
	return msg
}

// TPluralCount translates a message with plural support
func (s *I18nService) TPluralCount(localizer *i18n.Localizer, messageID string, count int, data map[string]interface{}) string {
	if data == nil {
		data = map[string]interface{}{}
	}
	data["Count"] = count

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
		PluralCount:  count,
	})
	if err != nil {
		return fmt.Sprintf("%s (%d)", messageID, count)
	}
	return msg
}

// SupportedLanguages returns the list of supported language codes
func (s *I18nService) SupportedLanguages() []string {
	return s.supportedLangs
}

// DefaultLanguage returns the default language code
func (s *I18nService) DefaultLanguage() string {
	return s.defaultLang
}
