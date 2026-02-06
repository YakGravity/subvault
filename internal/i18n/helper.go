package i18n

import (
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// TranslationHelper provides template-friendly translation methods
type TranslationHelper struct {
	localizer  *i18n.Localizer
	service    *I18nService
	lang       string
	dateFormat string // Go format string, e.g. "02.01.2006"
}

// NewTranslationHelper creates a new TranslationHelper for use in templates
func NewTranslationHelper(service *I18nService, localizer *i18n.Localizer, lang string) *TranslationHelper {
	return &TranslationHelper{
		localizer: localizer,
		service:   service,
		lang:      lang,
	}
}

// SetDateFormat overrides the locale-based date format with a custom Go format string
func (h *TranslationHelper) SetDateFormat(format string) {
	h.dateFormat = format
}

// FormatDate formats a date according to the current locale.
// Accepts time.Time or *time.Time.
func (h *TranslationHelper) FormatDate(v any) string {
	var t time.Time
	switch val := v.(type) {
	case time.Time:
		t = val
	case *time.Time:
		if val == nil {
			return ""
		}
		t = *val
	default:
		return ""
	}
	if h.dateFormat != "" {
		return t.Format(h.dateFormat)
	}
	switch h.lang {
	case "de":
		return t.Format("02.01.2006")
	default:
		return t.Format("Jan 2, 2006")
	}
}

// Tr translates a simple string by message ID
func (h *TranslationHelper) Tr(messageID string) string {
	return h.service.T(h.localizer, messageID)
}

// TrData translates a string with template data
func (h *TranslationHelper) TrData(messageID string, data map[string]interface{}) string {
	return h.service.TData(h.localizer, messageID, data)
}

// TrCount translates a string with plural support
func (h *TranslationHelper) TrCount(messageID string, count int) string {
	return h.service.TPluralCount(h.localizer, messageID, count, nil)
}

// TrCountData translates a string with plural support and template data
func (h *TranslationHelper) TrCountData(messageID string, count int, data map[string]interface{}) string {
	return h.service.TPluralCount(h.localizer, messageID, count, data)
}
