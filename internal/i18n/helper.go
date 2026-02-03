package i18n

import (
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// TranslationHelper provides template-friendly translation methods
type TranslationHelper struct {
	localizer *i18n.Localizer
	service   *I18nService
}

// NewTranslationHelper creates a new TranslationHelper for use in templates
func NewTranslationHelper(service *I18nService, localizer *i18n.Localizer) *TranslationHelper {
	return &TranslationHelper{
		localizer: localizer,
		service:   service,
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
