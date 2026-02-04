package service

import (
	"log"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"time"
)

type SubscriptionService struct {
	repo            *repository.SubscriptionRepository
	categoryService *CategoryService
	currencyService *CurrencyService
	settingsService *SettingsService
}

func NewSubscriptionService(repo *repository.SubscriptionRepository, categoryService *CategoryService, currencyService *CurrencyService, settingsService *SettingsService) *SubscriptionService {
	return &SubscriptionService{
		repo:            repo,
		categoryService: categoryService,
		currencyService: currencyService,
		settingsService: settingsService,
	}
}

func (s *SubscriptionService) Create(subscription *models.Subscription) (*models.Subscription, error) {
	return s.repo.Create(subscription)
}

func (s *SubscriptionService) GetAll() ([]models.Subscription, error) {
	return s.repo.GetAll()
}

func (s *SubscriptionService) GetAllSorted(sortBy, order string) ([]models.Subscription, error) {
	return s.repo.GetAllSorted(sortBy, order)
}

func (s *SubscriptionService) GetByID(id uint) (*models.Subscription, error) {
	return s.repo.GetByID(id)
}

func (s *SubscriptionService) Update(id uint, subscription *models.Subscription) (*models.Subscription, error) {
	return s.repo.Update(id, subscription)
}

func (s *SubscriptionService) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *SubscriptionService) Count() int64 {
	return s.repo.Count()
}

// convertAmount converts an amount from one currency to the display currency.
// Returns the original amount as fallback if conversion fails (e.g. no ECB rate for RUB/COP/BDT).
func (s *SubscriptionService) convertAmount(amount float64, fromCurrency, toCurrency string) float64 {
	if fromCurrency == toCurrency {
		return amount
	}
	converted, err := s.currencyService.ConvertAmount(amount, fromCurrency, toCurrency)
	if err != nil {
		// Fallback: use original amount (1:1) if conversion is not available
		return amount
	}
	return converted
}

func (s *SubscriptionService) GetStats() (*models.Stats, error) {
	displayCurrency := s.settingsService.GetCurrency()

	activeSubscriptions, err := s.repo.GetActiveSubscriptions()
	if err != nil {
		return nil, err
	}

	cancelledSubscriptions, err := s.repo.GetCancelledSubscriptions()
	if err != nil {
		return nil, err
	}

	upcomingRenewals, err := s.repo.GetUpcomingRenewals(7)
	if err != nil {
		return nil, err
	}

	stats := &models.Stats{
		ActiveSubscriptions:    len(activeSubscriptions),
		CancelledSubscriptions: len(cancelledSubscriptions),
		UpcomingRenewals:       len(upcomingRenewals),
		CategorySpending:       make(map[string]float64),
	}

	// Calculate totals with currency conversion
	for _, sub := range activeSubscriptions {
		monthly := s.convertAmount(sub.MonthlyCost(), sub.OriginalCurrency, displayCurrency)
		annual := s.convertAmount(sub.AnnualCost(), sub.OriginalCurrency, displayCurrency)
		stats.TotalMonthlySpend += monthly
		stats.TotalAnnualSpend += annual

		// Build category spending from active subscriptions (replaces SQL aggregation)
		categoryName := "Uncategorized"
		if sub.Category.Name != "" {
			categoryName = sub.Category.Name
		}
		stats.CategorySpending[categoryName] += monthly
	}

	// Calculate savings from cancelled subscriptions
	for _, sub := range cancelledSubscriptions {
		stats.TotalSaved += s.convertAmount(sub.AnnualCost(), sub.OriginalCurrency, displayCurrency)
		stats.MonthlySaved += s.convertAmount(sub.MonthlyCost(), sub.OriginalCurrency, displayCurrency)
	}

	// Log if any subscriptions couldn't be converted
	for _, sub := range activeSubscriptions {
		if sub.OriginalCurrency != displayCurrency && !HasECBRate(sub.OriginalCurrency) {
			log.Printf("Warning: No ECB exchange rate for %s, using 1:1 fallback for subscription %q", sub.OriginalCurrency, sub.Name)
		}
	}

	return stats, nil
}

func (s *SubscriptionService) GetAllCategories() ([]models.Category, error) {
	return s.categoryService.GetAll()
}

func (s *SubscriptionService) GetDefaultCategory() (*models.Category, error) {
	return s.categoryService.GetDefault()
}

// GetSubscriptionsNeedingReminders returns subscriptions that need renewal reminders
// based on the reminder_days setting. It returns a map of subscription to days until renewal.
func (s *SubscriptionService) GetSubscriptionsNeedingReminders(reminderDays int) (map[*models.Subscription]int, error) {
	if reminderDays <= 0 {
		return make(map[*models.Subscription]int), nil
	}

	// Get all subscriptions with renewals in the next reminderDays
	subscriptions, err := s.repo.GetUpcomingRenewals(reminderDays)
	if err != nil {
		return nil, err
	}

	result := make(map[*models.Subscription]int)

	for i := range subscriptions {
		sub := &subscriptions[i]
		if sub.RenewalDate == nil {
			continue
		}

		// Calculate days until renewal using proper date arithmetic
		// Use time.Until for more accurate calculation (handles timezone differences better)
		daysUntil := int(time.Until(*sub.RenewalDate).Hours() / 24)

		// Only include if within the reminder window and not past due
		if daysUntil >= 0 && daysUntil <= reminderDays {
			// Check if we've already sent a reminder for this renewal date
			// Skip if we've sent a reminder for the same renewal date
			if sub.LastReminderRenewalDate != nil &&
				sub.RenewalDate != nil &&
				sub.LastReminderRenewalDate.Equal(*sub.RenewalDate) {
				// Already sent reminder for this renewal date, skip
				continue
			}

			result[sub] = daysUntil
		}
	}

	return result, nil
}

// GetSubscriptionsNeedingCancellationReminders returns subscriptions that need cancellation reminders
// based on the cancellation_reminder_days setting. It returns a map of subscription to days until cancellation.
func (s *SubscriptionService) GetSubscriptionsNeedingCancellationReminders(reminderDays int) (map[*models.Subscription]int, error) {
	if reminderDays <= 0 {
		return make(map[*models.Subscription]int), nil
	}

	// Get all subscriptions with cancellations in the next reminderDays
	subscriptions, err := s.repo.GetUpcomingCancellations(reminderDays)
	if err != nil {
		return nil, err
	}

	result := make(map[*models.Subscription]int)

	for i := range subscriptions {
		sub := &subscriptions[i]
		if sub.CancellationDate == nil {
			continue
		}

		// Calculate days until cancellation
		daysUntil := int(time.Until(*sub.CancellationDate).Hours() / 24)

		// Only include if within the reminder window and not past due
		if daysUntil >= 0 && daysUntil <= reminderDays {
			// Check if we've already sent a reminder for this cancellation date
			if sub.LastCancellationReminderDate != nil &&
				sub.CancellationDate != nil &&
				sub.LastCancellationReminderDate.Equal(*sub.CancellationDate) {
				// Already sent reminder for this cancellation date, skip
				continue
			}

			result[sub] = daysUntil
		}
	}

	return result, nil
}
