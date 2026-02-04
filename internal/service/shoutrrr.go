package service

import (
	"fmt"
	"log"
	"strings"
	"subtrackr/internal/i18n"
	"subtrackr/internal/models"

	"github.com/containrrr/shoutrrr"
	t "github.com/containrrr/shoutrrr/pkg/types"
)

type ShoutrrrService struct {
	settingsService *SettingsService
	i18nService     *i18n.I18nService
}

func NewShoutrrrService(settingsService *SettingsService, i18nService ...*i18n.I18nService) *ShoutrrrService {
	svc := &ShoutrrrService{
		settingsService: settingsService,
	}
	if len(i18nService) > 0 {
		svc.i18nService = i18nService[0]
	}
	return svc
}

func (s *ShoutrrrService) t(messageID string) string {
	if s.i18nService == nil {
		return messageID
	}
	lang := s.settingsService.GetLanguage()
	localizer := s.i18nService.NewLocalizer(lang)
	return s.i18nService.T(localizer, messageID)
}

func (s *ShoutrrrService) tPlural(messageID string, count int, data map[string]interface{}) string {
	if s.i18nService == nil {
		return messageID
	}
	lang := s.settingsService.GetLanguage()
	localizer := s.i18nService.NewLocalizer(lang)
	return s.i18nService.TPluralCount(localizer, messageID, count, data)
}

func (s *ShoutrrrService) sendToAll(title, message string) error {
	config, err := s.settingsService.GetShoutrrrConfig()
	if err != nil {
		return fmt.Errorf("failed to get Shoutrrr config: %w", err)
	}

	if len(config.URLs) == 0 {
		return fmt.Errorf("Shoutrrr not configured: no notification URLs defined")
	}

	sender, err := shoutrrr.CreateSender(config.URLs...)
	if err != nil {
		return fmt.Errorf("failed to create Shoutrrr sender: %w", err)
	}

	params := t.Params{}
	if title != "" {
		params["title"] = title
	}

	errs := sender.Send(message, &params)

	var errMsgs []string
	for _, e := range errs {
		if e != nil {
			errMsgs = append(errMsgs, e.Error())
		}
	}

	if len(errMsgs) > 0 {
		return fmt.Errorf("shoutrrr send errors: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// SendTestNotification sends a test notification to the given URLs
func (s *ShoutrrrService) SendTestNotification(urls []string) error {
	if len(urls) == 0 {
		return fmt.Errorf("no notification URLs provided")
	}

	sender, err := shoutrrr.CreateSender(urls...)
	if err != nil {
		return fmt.Errorf("failed to create Shoutrrr sender: %w", err)
	}

	params := t.Params{
		"title": "SubTrackr Test",
	}
	errs := sender.Send("This is a test notification from SubTrackr. If you received this, your notification configuration is working correctly!", &params)

	var errMsgs []string
	for _, e := range errs {
		if e != nil {
			errMsgs = append(errMsgs, e.Error())
		}
	}

	if len(errMsgs) > 0 {
		return fmt.Errorf("shoutrrr send errors: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

func (s *ShoutrrrService) SendHighCostAlert(subscription *models.Subscription) error {
	currencySymbol := s.settingsService.GetCurrencySymbol()

	message := fmt.Sprintf("⚠️ %s\n\n", s.t("shoutrrr_high_cost_alert"))
	message += fmt.Sprintf("%s %s\n", s.t("email_name"), subscription.Name)
	message += fmt.Sprintf("%s %s%.2f %s\n", s.t("shoutrrr_cost"), currencySymbol, subscription.Cost, subscription.Schedule)
	message += fmt.Sprintf("%s %s%.2f\n", s.t("shoutrrr_monthly_cost"), currencySymbol, subscription.MonthlyCost())
	if subscription.Category.Name != "" {
		message += fmt.Sprintf("%s %s\n", s.t("shoutrrr_category"), subscription.Category.Name)
	}
	if subscription.RenewalDate != nil {
		message += fmt.Sprintf("%s %s\n", s.t("shoutrrr_next_renewal"), subscription.RenewalDate.Format("January 2, 2006"))
	}
	if subscription.URL != "" {
		message += fmt.Sprintf("%s %s", s.t("shoutrrr_url"), subscription.URL)
	}

	title := fmt.Sprintf("%s: %s", s.t("shoutrrr_high_cost_alert"), subscription.Name)

	if err := s.sendToAll(title, message); err != nil {
		log.Printf("Failed to send high cost alert via Shoutrrr: %v", err)
		return err
	}
	return nil
}

func (s *ShoutrrrService) SendRenewalReminder(subscription *models.Subscription, daysUntilRenewal int) error {
	currencySymbol := s.settingsService.GetCurrencySymbol()
	renewalText := s.tPlural("email_renewal_reminder", daysUntilRenewal, map[string]interface{}{"Name": subscription.Name})

	message := fmt.Sprintf("🔔 %s\n\n", s.t("shoutrrr_renewal_reminder"))
	message += renewalText + "\n\n"
	message += s.t("shoutrrr_sub_details") + "\n"
	message += fmt.Sprintf("%s %s%.2f %s\n", s.t("shoutrrr_cost"), currencySymbol, subscription.Cost, subscription.Schedule)
	message += fmt.Sprintf("%s %s%.2f\n", s.t("shoutrrr_monthly_cost"), currencySymbol, subscription.MonthlyCost())
	if subscription.Category.Name != "" {
		message += fmt.Sprintf("%s %s\n", s.t("shoutrrr_category"), subscription.Category.Name)
	}
	if subscription.RenewalDate != nil {
		message += fmt.Sprintf("%s %s\n", s.t("shoutrrr_renewal_date"), subscription.RenewalDate.Format("January 2, 2006"))
	}
	if subscription.URL != "" {
		message += fmt.Sprintf("%s %s", s.t("shoutrrr_url"), subscription.URL)
	}

	title := fmt.Sprintf("%s: %s", s.t("shoutrrr_renewal_reminder"), subscription.Name)

	if err := s.sendToAll(title, message); err != nil {
		log.Printf("Failed to send renewal reminder via Shoutrrr: %v", err)
		return err
	}
	return nil
}

func (s *ShoutrrrService) SendCancellationReminder(subscription *models.Subscription, daysUntilCancellation int) error {
	currencySymbol := s.settingsService.GetCurrencySymbol()
	cancellationText := s.tPlural("email_cancellation_reminder", daysUntilCancellation, map[string]interface{}{"Name": subscription.Name})

	message := fmt.Sprintf("⚠️ %s\n\n", s.t("shoutrrr_cancellation_reminder"))
	message += cancellationText + "\n\n"
	message += s.t("shoutrrr_sub_details") + "\n"
	message += fmt.Sprintf("%s %s%.2f %s\n", s.t("shoutrrr_cost"), currencySymbol, subscription.Cost, subscription.Schedule)
	message += fmt.Sprintf("%s %s%.2f\n", s.t("shoutrrr_monthly_cost"), currencySymbol, subscription.MonthlyCost())
	if subscription.Category.Name != "" {
		message += fmt.Sprintf("%s %s\n", s.t("shoutrrr_category"), subscription.Category.Name)
	}
	if subscription.CancellationDate != nil {
		message += fmt.Sprintf("%s %s\n", s.t("shoutrrr_cancellation_date"), subscription.CancellationDate.Format("January 2, 2006"))
	}
	if subscription.URL != "" {
		message += fmt.Sprintf("%s %s", s.t("shoutrrr_url"), subscription.URL)
	}

	title := fmt.Sprintf("%s: %s", s.t("shoutrrr_cancellation_reminder"), subscription.Name)

	if err := s.sendToAll(title, message); err != nil {
		log.Printf("Failed to send cancellation reminder via Shoutrrr: %v", err)
		return err
	}
	return nil
}

func (s *ShoutrrrService) SendBudgetExceededAlert(totalSpend, budget float64, currencySymbol string) error {
	message := fmt.Sprintf("%s\n%s: %s%.2f\n%s: %s%.2f\n%s: %s%.2f",
		s.t("budget_exceeded_alert"),
		s.t("dashboard_budget"), currencySymbol, budget,
		s.t("analytics_monthly_cost"), currencySymbol, totalSpend,
		s.t("dashboard_budget_exceeded"), currencySymbol, totalSpend-budget,
	)

	title := s.t("shoutrrr_budget_exceeded")

	if err := s.sendToAll(title, message); err != nil {
		log.Printf("Failed to send budget exceeded alert via Shoutrrr: %v", err)
		return err
	}
	return nil
}
