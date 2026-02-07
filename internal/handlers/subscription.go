package handlers

import (
	"subtrackr/internal/models"
	"subtrackr/internal/service"
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
	settingsService service.SettingsServiceInterface
	currencyService service.CurrencyServiceInterface
	emailService    service.EmailServiceInterface
	shoutrrrService service.ShoutrrrServiceInterface
	logoService     service.LogoServiceInterface
}

func NewSubscriptionHandler(service service.SubscriptionServiceInterface, settingsService service.SettingsServiceInterface, currencyService service.CurrencyServiceInterface, emailService service.EmailServiceInterface, shoutrrrService service.ShoutrrrServiceInterface, logoService service.LogoServiceInterface) *SubscriptionHandler {
	return &SubscriptionHandler{
		service:         service,
		settingsService: settingsService,
		currencyService: currencyService,
		emailService:    emailService,
		shoutrrrService: shoutrrrService,
		logoService:     logoService,
	}
}
