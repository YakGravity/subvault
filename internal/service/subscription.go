package service

import (
	"log/slog"
	"subvault/internal/models"
	"subvault/internal/repository"
	"time"
)

type SubscriptionService struct {
	repo            *repository.SubscriptionRepository
	categoryService *CategoryService
	currencyService *CurrencyService
	preferences     PreferencesServiceInterface
	settings        SettingsServiceInterface
	renewalService  RenewalServiceInterface
}

func NewSubscriptionService(repo *repository.SubscriptionRepository, categoryService *CategoryService, currencyService *CurrencyService, preferences PreferencesServiceInterface, settings SettingsServiceInterface, renewalService RenewalServiceInterface) *SubscriptionService {
	return &SubscriptionService{
		repo:            repo,
		categoryService: categoryService,
		currencyService: currencyService,
		preferences:     preferences,
		settings:        settings,
		renewalService:  renewalService,
	}
}

func (s *SubscriptionService) Create(subscription *models.Subscription) (*models.Subscription, error) {
	s.renewalService.InitializeRenewalDate(subscription)
	return s.repo.Create(subscription)
}

func (s *SubscriptionService) GetAll() ([]models.Subscription, error) {
	return s.repo.GetAll()
}

func (s *SubscriptionService) GetAllPaginated(limit, offset int) ([]models.Subscription, int64, error) {
	return s.repo.GetAllPaginated(limit, offset)
}

func (s *SubscriptionService) GetAllSorted(sortBy, order string) ([]models.Subscription, error) {
	return s.repo.GetAllSorted(sortBy, order)
}

func (s *SubscriptionService) GetByID(id uint) (*models.Subscription, error) {
	return s.repo.GetByID(id)
}

func (s *SubscriptionService) Update(id uint, subscription *models.Subscription) (*models.Subscription, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	s.renewalService.RecalculateIfNeeded(existing, subscription)
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
	displayCurrency := s.preferences.GetCurrency()

	// Single query: load all subscriptions with categories
	allSubs, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}

	// Partition in-memory
	now := time.Now()
	renewalCutoff := now.AddDate(0, 0, 7)

	stats := &models.Stats{
		CategorySpending: make(map[string]float64),
		AllSubscriptions: allSubs,
	}

	for _, sub := range allSubs {
		switch sub.Status {
		case "Active":
			stats.ActiveSubscriptions++
			monthly := s.convertAmount(sub.MonthlyCost(), sub.OriginalCurrency, displayCurrency)
			annual := s.convertAmount(sub.AnnualCost(), sub.OriginalCurrency, displayCurrency)
			stats.TotalMonthlySpend += monthly
			stats.TotalAnnualSpend += annual

			categoryName := "Uncategorized"
			if sub.Category.Name != "" {
				categoryName = sub.Category.Name
			}
			stats.CategorySpending[categoryName] += monthly

			// Check upcoming renewals
			if sub.RenewalDate != nil && !sub.RenewalDate.Before(now) && !sub.RenewalDate.After(renewalCutoff) {
				stats.UpcomingRenewals++
			}

			if sub.OriginalCurrency != displayCurrency && !HasECBRate(sub.OriginalCurrency) {
				slog.Warn("no ECB exchange rate, using 1:1 fallback", "currency", sub.OriginalCurrency, "subscription", sub.Name)
			}
		case "Cancelled":
			stats.CancelledSubscriptions++
			stats.TotalSaved += s.convertAmount(sub.AnnualCost(), sub.OriginalCurrency, displayCurrency)
			stats.MonthlySaved += s.convertAmount(sub.MonthlyCost(), sub.OriginalCurrency, displayCurrency)
		}
	}

	// Budget calculation
	budget := s.settings.GetFloatSettingWithDefault("monthly_budget", 0)
	stats.MonthlyBudget = budget
	if budget > 0 {
		stats.BudgetUtilization = stats.TotalMonthlySpend / budget * 100
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
// based on per-subscription settings. It returns a map of subscription to days until renewal.
func (s *SubscriptionService) GetSubscriptionsNeedingReminders() (map[*models.Subscription]int, error) {
	subscriptions, err := s.repo.GetSubscriptionsWithRenewalReminder()
	if err != nil {
		return nil, err
	}

	result := make(map[*models.Subscription]int)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for i := range subscriptions {
		sub := &subscriptions[i]
		if sub.RenewalDate == nil || sub.RenewalReminderDays <= 0 {
			continue
		}

		renewalDay := time.Date(sub.RenewalDate.Year(), sub.RenewalDate.Month(), sub.RenewalDate.Day(), 0, 0, 0, 0, sub.RenewalDate.Location())
		daysUntil := int(renewalDay.Sub(today).Hours() / 24)

		if daysUntil >= 0 && daysUntil <= sub.RenewalReminderDays {
			if sub.LastReminderRenewalDate != nil &&
				sub.RenewalDate != nil &&
				sub.LastReminderRenewalDate.Equal(*sub.RenewalDate) {
				continue
			}

			result[sub] = daysUntil
		}
	}

	return result, nil
}

// GetSubscriptionsNeedingCancellationReminders returns subscriptions that need cancellation reminders
// based on per-subscription settings. It returns a map of subscription to days until cancellation.
func (s *SubscriptionService) GetSubscriptionsNeedingCancellationReminders() (map[*models.Subscription]int, error) {
	subscriptions, err := s.repo.GetSubscriptionsWithCancellationReminder()
	if err != nil {
		return nil, err
	}

	result := make(map[*models.Subscription]int)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for i := range subscriptions {
		sub := &subscriptions[i]
		if sub.CancellationDate == nil || sub.CancellationReminderDays <= 0 {
			continue
		}

		cancellationDay := time.Date(sub.CancellationDate.Year(), sub.CancellationDate.Month(), sub.CancellationDate.Day(), 0, 0, 0, 0, sub.CancellationDate.Location())
		daysUntil := int(cancellationDay.Sub(today).Hours() / 24)

		if daysUntil >= 0 && daysUntil <= sub.CancellationReminderDays {
			if sub.LastCancellationReminderDate != nil &&
				sub.CancellationDate != nil &&
				sub.LastCancellationReminderDate.Equal(*sub.CancellationDate) {
				continue
			}

			result[sub] = daysUntil
		}
	}

	return result, nil
}
