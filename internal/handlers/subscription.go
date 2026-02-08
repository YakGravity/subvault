package handlers

import (
	"subvault/internal/models"
	"subvault/internal/service"
)

// SubscriptionWithConversion represents a subscription with currency conversion info
type SubscriptionWithConversion struct {
	*models.Subscription
	ConvertedCost          float64 `json:"converted_cost"`
	ConvertedAnnualCost    float64 `json:"converted_annual_cost"`
	ConvertedMonthlyCost   float64 `json:"converted_monthly_cost"`
	DisplayCurrency        string  `json:"display_currency"`
	DisplayCurrencySymbol  string  `json:"display_currency_symbol"`
	OriginalCurrencySymbol string  `json:"original_currency_symbol"`
	ShowConversion         bool    `json:"show_conversion"`
}

type SubscriptionHandler struct {
	service         service.SubscriptionServiceInterface
	preferences     service.PreferencesServiceInterface
	settings        service.SettingsServiceInterface
	calendarService service.CalendarServiceInterface
	currencyService service.CurrencyServiceInterface
	emailService    service.EmailServiceInterface
	shoutrrrService service.ShoutrrrServiceInterface
	logoService     service.LogoServiceInterface
}

func NewSubscriptionHandler(svc service.SubscriptionServiceInterface, preferences service.PreferencesServiceInterface, settings service.SettingsServiceInterface, calendarService service.CalendarServiceInterface, currencyService service.CurrencyServiceInterface, emailService service.EmailServiceInterface, shoutrrrService service.ShoutrrrServiceInterface, logoService service.LogoServiceInterface) *SubscriptionHandler {
	return &SubscriptionHandler{
		service:         svc,
		preferences:     preferences,
		settings:        settings,
		calendarService: calendarService,
		currencyService: currencyService,
		emailService:    emailService,
		shoutrrrService: shoutrrrService,
		logoService:     logoService,
	}
}
