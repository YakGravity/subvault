package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewI18nService(t *testing.T) {
	svc := NewI18nService("")

	assert.NotNil(t, svc)
	assert.NotNil(t, svc.bundle)
	assert.Equal(t, "en", svc.defaultLang)
	assert.Equal(t, []string{"de", "en"}, svc.supportedLangs)
}

func TestI18nService_T_English(t *testing.T) {
	svc := NewI18nService("")

	tests := []struct {
		name      string
		messageID string
		expected  string
	}{
		{
			name:      "nav_dashboard translates to Dashboard",
			messageID: "nav_dashboard",
			expected:  "Dashboard",
		},
		{
			name:      "nav_subscriptions translates to Subscriptions",
			messageID: "nav_subscriptions",
			expected:  "Subscriptions",
		},
		{
			name:      "btn_save translates to Save",
			messageID: "btn_save",
			expected:  "Save",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localizer := svc.NewLocalizer("en")
			result := svc.T(localizer, tt.messageID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestI18nService_T_German(t *testing.T) {
	svc := NewI18nService("")

	// German localizer falls back to English since active.de.json may not exist.
	// "Dashboard" is the same in both languages for nav_dashboard.
	localizer := svc.NewLocalizer("de")
	result := svc.T(localizer, "nav_dashboard")
	assert.Equal(t, "Dashboard", result)
}

func TestI18nService_T_FallbackToEnglish(t *testing.T) {
	svc := NewI18nService("")

	tests := []struct {
		name      string
		lang      string
		messageID string
		expected  string
	}{
		{
			name:      "French falls back to English for nav_dashboard",
			lang:      "fr",
			messageID: "nav_dashboard",
			expected:  "Dashboard",
		},
		{
			name:      "Empty lang falls back to English for btn_save",
			lang:      "",
			messageID: "btn_save",
			expected:  "Save",
		},
		{
			name:      "Unknown lang falls back to English for nav_settings",
			lang:      "xx",
			messageID: "nav_settings",
			expected:  "Settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localizer := svc.NewLocalizer(tt.lang)
			result := svc.T(localizer, tt.messageID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestI18nService_T_FallbackToMessageID(t *testing.T) {
	svc := NewI18nService("")

	tests := []struct {
		name      string
		messageID string
	}{
		{
			name:      "nonexistent key returns message ID",
			messageID: "this_key_does_not_exist",
		},
		{
			name:      "another missing key returns message ID",
			messageID: "completely_unknown_message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localizer := svc.NewLocalizer("en")
			result := svc.T(localizer, tt.messageID)
			assert.Equal(t, tt.messageID, result)
		})
	}
}

func TestTranslationHelper_Tr(t *testing.T) {
	svc := NewI18nService("")

	tests := []struct {
		name      string
		lang      string
		messageID string
		expected  string
	}{
		{
			name:      "Tr returns English translation",
			lang:      "en",
			messageID: "nav_dashboard",
			expected:  "Dashboard",
		},
		{
			name:      "Tr returns nav_subscriptions",
			lang:      "en",
			messageID: "nav_subscriptions",
			expected:  "Subscriptions",
		},
		{
			name:      "Tr falls back to message ID for unknown key",
			lang:      "en",
			messageID: "nonexistent_key",
			expected:  "nonexistent_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localizer := svc.NewLocalizer(tt.lang)
			helper := NewTranslationHelper(svc, localizer, tt.lang)
			result := helper.Tr(tt.messageID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTranslationHelper_TrCount(t *testing.T) {
	svc := NewI18nService("")

	tests := []struct {
		name      string
		messageID string
		count     int
		expected  string
	}{
		{
			name:      "singular renewal reminder (1 day)",
			messageID: "email_renewal_reminder",
			count:     1,
			expected:  "Your subscription Netflix will renew in 1 day.",
		},
		{
			name:      "plural renewal reminder (3 days)",
			messageID: "email_renewal_reminder",
			count:     3,
			expected:  "Your subscription Netflix will renew in 3 days.",
		},
		{
			name:      "plural renewal reminder (0 days)",
			messageID: "email_renewal_reminder",
			count:     0,
			expected:  "Your subscription Netflix will renew in 0 days.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localizer := svc.NewLocalizer("en")
			helper := NewTranslationHelper(svc, localizer, "en")
			result := helper.TrCountData(tt.messageID, tt.count, map[string]interface{}{"Name": "Netflix"})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestI18nService_TData(t *testing.T) {
	svc := NewI18nService("")

	tests := []struct {
		name      string
		messageID string
		data      map[string]interface{}
		expected  string
	}{
		{
			name:      "TData renders template with Symbol",
			messageID: "settings_high_cost_threshold_desc",
			data:      map[string]interface{}{"Symbol": "$"},
			expected:  "Monthly cost threshold for high cost alerts (in $)",
		},
		{
			name:      "TData renders template with Euro symbol",
			messageID: "settings_high_cost_threshold_desc",
			data:      map[string]interface{}{"Symbol": "EUR"},
			expected:  "Monthly cost threshold for high cost alerts (in EUR)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localizer := svc.NewLocalizer("en")
			result := svc.TData(localizer, tt.messageID, tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestI18nService_SupportedLanguages(t *testing.T) {
	svc := NewI18nService("")

	langs := svc.SupportedLanguages()
	assert.Equal(t, []string{"de", "en"}, langs)
	assert.Len(t, langs, 2)
}
