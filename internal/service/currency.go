package service

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"sync"
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
	repo     *repository.ExchangeRateRepository
	mu       sync.RWMutex
	eurRates map[string]float64 // currency -> rate (EUR-based)
	rateDate time.Time
}

func NewCurrencyService(repo *repository.ExchangeRateRepository) *CurrencyService {
	return &CurrencyService{
		repo:     repo,
		eurRates: make(map[string]float64),
	}
}

// ensureRates loads exchange rates into memory if needed
func (s *CurrencyService) ensureRates() error {
	s.mu.RLock()
	if len(s.eurRates) > 0 && time.Since(s.rateDate) < 24*time.Hour {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after write lock
	if len(s.eurRates) > 0 && time.Since(s.rateDate) < 24*time.Hour {
		return nil
	}

	// Try loading from DB first
	rates, err := s.repo.GetLatestRates("EUR")
	if err == nil && len(rates) > 0 && !rates[0].IsStale() {
		s.eurRates = make(map[string]float64, len(rates)+1)
		s.eurRates["EUR"] = 1.0
		for _, r := range rates {
			s.eurRates[r.Currency] = r.Rate
		}
		s.rateDate = rates[0].Date
		return nil
	}

	// Fetch fresh rates from ECB
	return s.fetchAndCacheRatesLocked()
}

// getCrossRate computes a cross-rate via EUR. Caller must ensure rates are loaded.
func (s *CurrencyService) getCrossRate(from, to string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if from == to {
		return 1.0, nil
	}

	fromRate, okFrom := s.eurRates[from]
	toRate, okTo := s.eurRates[to]

	if !okFrom || !okTo || fromRate == 0 {
		return 0, fmt.Errorf("no exchange rate available for %s to %s", from, to)
	}

	// Cross-rate via EUR: toRate / fromRate
	return toRate / fromRate, nil
}

// GetExchangeRate retrieves exchange rate between two currencies
func (s *CurrencyService) GetExchangeRate(fromCurrency, toCurrency string) (float64, error) {
	if fromCurrency == toCurrency {
		return 1.0, nil
	}

	if !HasECBRate(fromCurrency) || !HasECBRate(toCurrency) {
		return 0, fmt.Errorf("no exchange rate available for %s to %s (not provided by ECB)", fromCurrency, toCurrency)
	}

	if err := s.ensureRates(); err != nil {
		return 0, err
	}

	return s.getCrossRate(fromCurrency, toCurrency)
}

// ConvertAmount converts an amount from one currency to another
func (s *CurrencyService) ConvertAmount(amount float64, fromCurrency, toCurrency string) (float64, error) {
	rate, err := s.GetExchangeRate(fromCurrency, toCurrency)
	if err != nil {
		return 0, err
	}
	return amount * rate, nil
}

// fetchAndCacheRatesLocked fetches all EUR-based rates from ECB and populates the in-memory cache.
// Caller must hold s.mu write lock.
func (s *CurrencyService) fetchAndCacheRatesLocked() error {
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
		return fmt.Errorf("failed to fetch ECB exchange rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ECB API returned status %d", resp.StatusCode)
	}

	var envelope ecbEnvelope
	if err := xml.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("failed to decode ECB response: %w", err)
	}

	if len(envelope.Rates) == 0 {
		return fmt.Errorf("ECB response contained no rates")
	}

	// Populate in-memory cache
	rateDate := time.Now()
	s.eurRates = make(map[string]float64, len(envelope.Rates)+1)
	s.eurRates["EUR"] = 1.0
	for _, r := range envelope.Rates {
		s.eurRates[r.Currency] = r.Rate
	}
	s.rateDate = rateDate

	// Persist to DB for restart recovery
	var ratesToSave []models.ExchangeRate
	for currency, rate := range s.eurRates {
		ratesToSave = append(ratesToSave, models.ExchangeRate{
			BaseCurrency: "EUR",
			Currency:     currency,
			Rate:         rate,
			Date:         rateDate,
		})
	}
	if err := s.repo.SaveRates(ratesToSave); err != nil {
		slog.Warn("failed to cache exchange rates", "error", err)
	}

	return nil
}

// RefreshRates updates all exchange rates from the ECB
func (s *CurrencyService) RefreshRates() error {
	s.mu.Lock()
	err := s.fetchAndCacheRatesLocked()
	s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("failed to refresh rates: %w", err)
	}
	return s.repo.DeleteStaleRates(7 * 24 * time.Hour)
}
