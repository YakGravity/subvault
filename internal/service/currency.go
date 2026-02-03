package service

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"strings"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"time"
)

const ecbDailyURL = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"

// SupportedCurrencies defines the list of currencies supported for exchange rates and settings.
// Currencies with ECB rates are listed first, followed by currencies without ECB data.
var SupportedCurrencies = []string{
	"EUR", "USD", "GBP", "JPY", "CHF", "SEK", "PLN", "INR", "BRL",
	"AUD", "CAD", "CNY", "CZK", "DKK", "HKD", "HUF", "IDR", "ILS",
	"ISK", "KRW", "MXN", "MYR", "NOK", "NZD", "PHP", "RON", "SGD",
	"THB", "TRY", "ZAR",
	"RUB", "COP", "BDT", // No ECB rates available for these
}

// ecbCurrencies contains currencies provided by the ECB (EUR is implicit as base)
var ecbCurrencies = map[string]bool{
	"USD": true, "JPY": true, "GBP": true, "CHF": true, "SEK": true,
	"PLN": true, "INR": true, "BRL": true, "AUD": true, "CAD": true,
	"CNY": true, "CZK": true, "DKK": true, "HKD": true, "HUF": true,
	"IDR": true, "ILS": true, "ISK": true, "KRW": true, "MXN": true,
	"MYR": true, "NOK": true, "NZD": true, "PHP": true, "RON": true,
	"SGD": true, "THB": true, "TRY": true, "ZAR": true,
}

// HasECBRate returns whether the ECB provides exchange rates for this currency
func HasECBRate(currency string) bool {
	if currency == "EUR" {
		return true
	}
	return ecbCurrencies[currency]
}

// supportedCurrencySymbols returns the currencies as a comma-separated string
func supportedCurrencySymbols() string {
	return strings.Join(SupportedCurrencies, ",")
}

// ECB XML response structs
type ecbEnvelope struct {
	XMLName xml.Name  `xml:"Envelope"`
	Rates   []ecbRate `xml:"Cube>Cube>Cube"`
}

type ecbRate struct {
	Currency string  `xml:"currency,attr"`
	Rate     float64 `xml:"rate,attr"`
}

type CurrencyService struct {
	repo *repository.ExchangeRateRepository
}

func NewCurrencyService(repo *repository.ExchangeRateRepository) *CurrencyService {
	return &CurrencyService{repo: repo}
}

// GetExchangeRate retrieves exchange rate between two currencies
func (s *CurrencyService) GetExchangeRate(fromCurrency, toCurrency string) (float64, error) {
	if fromCurrency == toCurrency {
		return 1.0, nil
	}

	// Try to get cached rate first
	rate, err := s.repo.GetRate(fromCurrency, toCurrency)
	if err == nil && !rate.IsStale() {
		return rate.Rate, nil
	}

	// Check if both currencies have ECB support
	if !HasECBRate(fromCurrency) || !HasECBRate(toCurrency) {
		return 0, fmt.Errorf("no exchange rate available for %s to %s (not provided by ECB)", fromCurrency, toCurrency)
	}

	// Fetch from ECB
	return s.fetchAndCacheRates(fromCurrency, toCurrency)
}

// ConvertAmount converts an amount from one currency to another
func (s *CurrencyService) ConvertAmount(amount float64, fromCurrency, toCurrency string) (float64, error) {
	rate, err := s.GetExchangeRate(fromCurrency, toCurrency)
	if err != nil {
		return 0, err
	}
	return amount * rate, nil
}

// fetchAndCacheRates fetches rates from the ECB and caches them.
// ECB provides rates with EUR as base, so cross-rate calculations are used for other pairs.
func (s *CurrencyService) fetchAndCacheRates(baseCurrency, targetCurrency string) (float64, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}
	resp, err := client.Get(ecbDailyURL)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch ECB exchange rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("ECB API returned status %d", resp.StatusCode)
	}

	var envelope ecbEnvelope
	if err := xml.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return 0, fmt.Errorf("failed to decode ECB response: %w", err)
	}

	if len(envelope.Rates) == 0 {
		return 0, fmt.Errorf("ECB response contained no rates")
	}

	// Build rates map
	ratesMap := make(map[string]float64)
	for _, r := range envelope.Rates {
		ratesMap[r.Currency] = r.Rate
	}

	// Cache all rates with EUR as base
	rateDate := time.Now()
	var ratesToSave []models.ExchangeRate

	ratesToSave = append(ratesToSave, models.ExchangeRate{
		BaseCurrency: "EUR",
		Currency:     "EUR",
		Rate:         1.0,
		Date:         rateDate,
	})

	for currency, rate := range ratesMap {
		ratesToSave = append(ratesToSave, models.ExchangeRate{
			BaseCurrency: "EUR",
			Currency:     currency,
			Rate:         rate,
			Date:         rateDate,
		})
	}

	if len(ratesToSave) > 0 {
		if err := s.repo.SaveRates(ratesToSave); err != nil {
			log.Printf("Warning: failed to cache exchange rates: %v", err)
		}
	}

	// Calculate cross-rate
	if baseCurrency == "EUR" {
		if rate, exists := ratesMap[targetCurrency]; exists {
			return rate, nil
		}
	} else if targetCurrency == "EUR" {
		if rate, exists := ratesMap[baseCurrency]; exists && rate != 0 {
			return 1.0 / rate, nil
		}
	} else {
		baseToEur, exists1 := ratesMap[baseCurrency]
		eurToTarget, exists2 := ratesMap[targetCurrency]
		if exists1 && exists2 && baseToEur != 0 {
			return eurToTarget / baseToEur, nil
		}
	}

	return 0, fmt.Errorf("exchange rate for %s to %s not available", baseCurrency, targetCurrency)
}

// RefreshRates updates all exchange rates from the ECB
func (s *CurrencyService) RefreshRates() error {
	_, err := s.fetchAndCacheRates("EUR", "USD")
	if err != nil {
		return fmt.Errorf("failed to refresh rates: %w", err)
	}
	return s.repo.DeleteStaleRates(7 * 24 * time.Hour)
}
