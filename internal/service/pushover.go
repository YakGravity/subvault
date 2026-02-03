package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"subtrackr/internal/i18n"
	"subtrackr/internal/models"
	"time"
)

// PushoverService handles sending notifications via Pushover
type PushoverService struct {
	settingsService *SettingsService
	i18nService     *i18n.I18nService
}

// NewPushoverService creates a new Pushover service
func NewPushoverService(settingsService *SettingsService, i18nService ...*i18n.I18nService) *PushoverService {
	svc := &PushoverService{
		settingsService: settingsService,
	}
	if len(i18nService) > 0 {
		svc.i18nService = i18nService[0]
	}
	return svc
}

// t translates a message ID using the user's language setting
func (p *PushoverService) t(messageID string) string {
	if p.i18nService == nil {
		return messageID
	}
	lang := p.settingsService.GetLanguage()
	localizer := p.i18nService.NewLocalizer(lang)
	return p.i18nService.T(localizer, messageID)
}

// tPlural translates a message ID with plural support
func (p *PushoverService) tPlural(messageID string, count int, data map[string]interface{}) string {
	if p.i18nService == nil {
		return messageID
	}
	lang := p.settingsService.GetLanguage()
	localizer := p.i18nService.NewLocalizer(lang)
	return p.i18nService.TPluralCount(localizer, messageID, count, data)
}

// PushoverResponse represents the response from Pushover API
type PushoverResponse struct {
	Status  int      `json:"status"`
	Request string   `json:"request"`
	Errors  []string `json:"errors,omitempty"`
}

// SendNotification sends a notification via Pushover
func (p *PushoverService) SendNotification(title, message string, priority int) error {
	config, err := p.settingsService.GetPushoverConfig()
	if err != nil {
		return fmt.Errorf("failed to get Pushover config: %w", err)
	}

	if config.UserKey == "" || config.AppToken == "" {
		return fmt.Errorf("Pushover not configured: user key and app token required")
	}

	// Pushover API endpoint
	apiURL := "https://api.pushover.net/1/messages.json"

	// Prepare form data
	formData := url.Values{}
	formData.Set("token", config.AppToken)
	formData.Set("user", config.UserKey)
	formData.Set("title", title)
	formData.Set("message", message)
	formData.Set("priority", strconv.Itoa(priority))

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Pushover notification: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var pushoverResp PushoverResponse
	if err := json.NewDecoder(resp.Body).Decode(&pushoverResp); err != nil {
		return fmt.Errorf("failed to decode Pushover response: %w", err)
	}

	if pushoverResp.Status != 1 {
		errorMsg := "Pushover API error"
		if len(pushoverResp.Errors) > 0 {
			errorMsg = pushoverResp.Errors[0]
		}
		return fmt.Errorf("%s", errorMsg)
	}

	return nil
}

// SendHighCostAlert sends a Pushover alert when a high-cost subscription is created
func (p *PushoverService) SendHighCostAlert(subscription *models.Subscription) error {
	// Check if high cost alerts are enabled
	enabled, err := p.settingsService.GetBoolSetting("high_cost_alerts", true)
	if err != nil || !enabled {
		return nil // Silently skip if disabled
	}

	// Get currency symbol
	currencySymbol := p.settingsService.GetCurrencySymbol()

	// Build message
	message := fmt.Sprintf("⚠️ %s\n\n", p.t("pushover_high_cost_alert"))
	message += fmt.Sprintf("%s %s\n", p.t("email_name"), subscription.Name)
	message += fmt.Sprintf("%s %s%.2f %s\n", p.t("pushover_cost"), currencySymbol, subscription.Cost, subscription.Schedule)
	message += fmt.Sprintf("%s %s%.2f\n", p.t("pushover_monthly_cost"), currencySymbol, subscription.MonthlyCost())
	if subscription.Category.Name != "" {
		message += fmt.Sprintf("%s %s\n", p.t("pushover_category"), subscription.Category.Name)
	}
	if subscription.RenewalDate != nil {
		message += fmt.Sprintf("%s %s\n", p.t("pushover_next_renewal"), subscription.RenewalDate.Format("January 2, 2006"))
	}
	if subscription.URL != "" {
		message += fmt.Sprintf("%s %s", p.t("pushover_url"), subscription.URL)
	}

	title := fmt.Sprintf("%s: %s", p.t("pushover_high_cost_alert"), subscription.Name)
	return p.SendNotification(title, message, 1)
}

// SendRenewalReminder sends a Pushover reminder for an upcoming subscription renewal
func (p *PushoverService) SendRenewalReminder(subscription *models.Subscription, daysUntilRenewal int) error {
	// Check if renewal reminders are enabled
	enabled, err := p.settingsService.GetBoolSetting("renewal_reminders", false)
	if err != nil || !enabled {
		return nil // Silently skip if disabled
	}

	// Get currency symbol
	currencySymbol := p.settingsService.GetCurrencySymbol()

	renewalText := p.tPlural("email_renewal_reminder", daysUntilRenewal, map[string]interface{}{"Name": subscription.Name})

	message := fmt.Sprintf("🔔 %s\n\n", p.t("pushover_renewal_reminder"))
	message += renewalText + "\n\n"
	message += p.t("pushover_sub_details") + "\n"
	message += fmt.Sprintf("%s %s%.2f %s\n", p.t("pushover_cost"), currencySymbol, subscription.Cost, subscription.Schedule)
	message += fmt.Sprintf("%s %s%.2f\n", p.t("pushover_monthly_cost"), currencySymbol, subscription.MonthlyCost())
	if subscription.Category.Name != "" {
		message += fmt.Sprintf("%s %s\n", p.t("pushover_category"), subscription.Category.Name)
	}
	if subscription.RenewalDate != nil {
		message += fmt.Sprintf("%s %s\n", p.t("pushover_renewal_date"), subscription.RenewalDate.Format("January 2, 2006"))
	}
	if subscription.URL != "" {
		message += fmt.Sprintf("%s %s", p.t("pushover_url"), subscription.URL)
	}

	title := fmt.Sprintf("%s: %s", p.t("pushover_renewal_reminder"), subscription.Name)
	return p.SendNotification(title, message, 0)
}

// SendCancellationReminder sends a Pushover reminder for an upcoming subscription cancellation
func (p *PushoverService) SendCancellationReminder(subscription *models.Subscription, daysUntilCancellation int) error {
	// Check if cancellation reminders are enabled
	enabled, err := p.settingsService.GetBoolSetting("cancellation_reminders", false)
	if err != nil || !enabled {
		return nil // Silently skip if disabled
	}

	// Get currency symbol
	currencySymbol := p.settingsService.GetCurrencySymbol()

	cancellationText := p.tPlural("email_cancellation_reminder", daysUntilCancellation, map[string]interface{}{"Name": subscription.Name})

	message := fmt.Sprintf("⚠️ %s\n\n", p.t("pushover_cancellation_reminder"))
	message += cancellationText + "\n\n"
	message += p.t("pushover_sub_details") + "\n"
	message += fmt.Sprintf("%s %s%.2f %s\n", p.t("pushover_cost"), currencySymbol, subscription.Cost, subscription.Schedule)
	message += fmt.Sprintf("%s %s%.2f\n", p.t("pushover_monthly_cost"), currencySymbol, subscription.MonthlyCost())
	if subscription.Category.Name != "" {
		message += fmt.Sprintf("%s %s\n", p.t("pushover_category"), subscription.Category.Name)
	}
	if subscription.CancellationDate != nil {
		message += fmt.Sprintf("%s %s\n", p.t("pushover_cancellation_date"), subscription.CancellationDate.Format("January 2, 2006"))
	}
	if subscription.URL != "" {
		message += fmt.Sprintf("%s %s", p.t("pushover_url"), subscription.URL)
	}

	title := fmt.Sprintf("%s: %s", p.t("pushover_cancellation_reminder"), subscription.Name)
	return p.SendNotification(title, message, 0)
}
