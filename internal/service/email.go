package service

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"subvault/internal/i18n"
	"subvault/internal/models"
)

// EmailService handles sending emails via SMTP
type EmailService struct {
	preferences PreferencesServiceInterface
	notifConfig NotificationConfigServiceInterface
	i18nService *i18n.I18nService
}

// NewEmailService creates a new email service
func NewEmailService(preferences PreferencesServiceInterface, notifConfig NotificationConfigServiceInterface, i18nService ...*i18n.I18nService) *EmailService {
	svc := &EmailService{
		preferences: preferences,
		notifConfig: notifConfig,
	}
	if len(i18nService) > 0 {
		svc.i18nService = i18nService[0]
	}
	return svc
}

// t translates a message ID using the user's language setting
func (e *EmailService) t(messageID string) string {
	if e.i18nService == nil {
		return messageID
	}
	lang := e.preferences.GetLanguage()
	localizer := e.i18nService.NewLocalizer(lang)
	return e.i18nService.T(localizer, messageID)
}

// tData translates a message ID with template data
func (e *EmailService) tData(messageID string, data map[string]interface{}) string {
	if e.i18nService == nil {
		return messageID
	}
	lang := e.preferences.GetLanguage()
	localizer := e.i18nService.NewLocalizer(lang)
	return e.i18nService.TData(localizer, messageID, data)
}

// tPlural translates a message ID with plural support
func (e *EmailService) tPlural(messageID string, count int, data map[string]interface{}) string {
	if e.i18nService == nil {
		return messageID
	}
	lang := e.preferences.GetLanguage()
	localizer := e.i18nService.NewLocalizer(lang)
	return e.i18nService.TPluralCount(localizer, messageID, count, data)
}

// SendEmail sends an email using the configured SMTP settings
func (e *EmailService) SendEmail(subject, body string) error {
	config, err := e.notifConfig.GetSMTPConfig()
	if err != nil {
		return fmt.Errorf("failed to get SMTP config: %w", err)
	}

	if config.To == "" {
		return fmt.Errorf("no recipient email configured")
	}

	// Determine if this is an implicit TLS port (SMTPS)
	isSSLPort := config.Port == 465 || config.Port == 8465 || config.Port == 443

	var auth smtp.Auth
	var addr string

	auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	addr = fmt.Sprintf("%s:%d", config.Host, config.Port)

	if isSSLPort {
		// Use implicit TLS (direct SSL connection)
		tlsConfig := &tls.Config{
			ServerName: config.Host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect via SSL: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, config.Host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()

		// Authenticate
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		// Set sender and recipient
		if err = client.Mail(config.From); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}
		if err = client.Rcpt(config.To); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}

		// Send email body
		writer, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %w", err)
		}

		fromName := config.FromName
		if fromName == "" {
			fromName = "SubVault"
		}

		message := fmt.Sprintf("From: %s <%s>\r\n", fromName, config.From)
		message += fmt.Sprintf("To: %s\r\n", config.To)
		message += fmt.Sprintf("Subject: %s\r\n", subject)
		message += "MIME-Version: 1.0\r\n"
		message += "Content-Type: text/html; charset=UTF-8\r\n"
		message += "\r\n"
		message += body

		_, err = writer.Write([]byte(message))
		if err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}
		err = writer.Close()
		if err != nil {
			return fmt.Errorf("failed to close writer: %w", err)
		}
	} else {
		// Use STARTTLS (opportunistic TLS)
		client, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer client.Close()

		// Upgrade to TLS
		tlsConfig := &tls.Config{
			ServerName: config.Host,
		}

		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}

		// Authenticate
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		// Set sender and recipient
		if err = client.Mail(config.From); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}
		if err = client.Rcpt(config.To); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}

		// Send email body
		writer, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %w", err)
		}

		fromName := config.FromName
		if fromName == "" {
			fromName = "SubVault"
		}

		message := fmt.Sprintf("From: %s <%s>\r\n", fromName, config.From)
		message += fmt.Sprintf("To: %s\r\n", config.To)
		message += fmt.Sprintf("Subject: %s\r\n", subject)
		message += "MIME-Version: 1.0\r\n"
		message += "Content-Type: text/html; charset=UTF-8\r\n"
		message += "\r\n"
		message += body

		_, err = writer.Write([]byte(message))
		if err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}
		err = writer.Close()
		if err != nil {
			return fmt.Errorf("failed to close writer: %w", err)
		}
	}

	return nil
}

// SendHighCostAlert sends an email alert when a high-cost subscription is created
func (e *EmailService) SendHighCostAlert(subscription *models.Subscription) error {
	// Get currency symbol
	currencySymbol := e.preferences.GetCurrencySymbol()

	// Build email body
	tmpl := `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.alert { background-color: #fff3cd; border: 1px solid #ffc107; border-radius: 5px; padding: 15px; margin: 20px 0; }
		.subscription-details { background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 20px 0; }
		.detail-row { margin: 10px 0; }
		.label { font-weight: bold; }
		.footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h2>{{.Title}}</h2>
		<div class="alert">
			<strong>` + "\u26a0\ufe0f" + ` {{.AlertLabel}}</strong> {{.AlertText}}
		</div>
		<div class="subscription-details">
			<h3>{{.DetailsTitle}}</h3>
			<div class="detail-row"><span class="label">{{.LabelName}}</span> {{.Subscription.Name}}</div>
			<div class="detail-row"><span class="label">{{.LabelCost}}</span> {{.CurrencySymbol}}{{printf "%.2f" .Subscription.Cost}} {{.Subscription.Schedule}}</div>
			<div class="detail-row"><span class="label">{{.LabelMonthlyCost}}</span> {{.CurrencySymbol}}{{printf "%.2f" (.Subscription.MonthlyCost)}}</div>
			{{if and .Subscription.Category .Subscription.Category.Name}}<div class="detail-row"><span class="label">{{.LabelCategory}}</span> {{.Subscription.Category.Name}}</div>{{end}}
			{{if .Subscription.RenewalDate}}<div class="detail-row"><span class="label">{{.LabelNextRenewal}}</span> {{.Subscription.RenewalDate.Format "January 2, 2006"}}</div>{{end}}
			{{if .Subscription.URL}}<div class="detail-row"><span class="label">{{.LabelURL}}</span> <a href="{{.Subscription.URL}}">{{.Subscription.URL}}</a></div>{{end}}
		</div>
		<div class="footer">
			<p>{{.FooterAuto}}</p>
			<p>{{.FooterManage}}</p>
		</div>
	</div>
</body>
</html>
`

	type AlertData struct {
		Subscription     *models.Subscription
		CurrencySymbol   string
		Title            string
		AlertLabel       string
		AlertText        string
		DetailsTitle     string
		LabelName        string
		LabelCost        string
		LabelMonthlyCost string
		LabelCategory    string
		LabelNextRenewal string
		LabelURL         string
		FooterAuto       string
		FooterManage     string
	}

	data := AlertData{
		Subscription:     subscription,
		CurrencySymbol:   currencySymbol,
		Title:            e.t("email_high_cost_title"),
		AlertLabel:       "Alert:",
		AlertText:        e.t("email_high_cost_alert"),
		DetailsTitle:     e.t("email_sub_details"),
		LabelName:        e.t("email_name"),
		LabelCost:        e.t("email_cost"),
		LabelMonthlyCost: e.t("email_monthly_cost"),
		LabelCategory:    e.t("email_category"),
		LabelNextRenewal: e.t("email_next_renewal"),
		LabelURL:         e.t("email_url"),
		FooterAuto:       e.t("email_footer_auto"),
		FooterManage:     e.t("email_footer_manage"),
	}

	tpl, err := template.New("highCostAlert").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	subject := fmt.Sprintf("%s: %s - %s%.2f/month", e.t("shoutrrr_high_cost_alert"), subscription.Name, currencySymbol, subscription.MonthlyCost())
	return e.SendEmail(subject, buf.String())
}

// SendRenewalReminder sends an email reminder for an upcoming subscription renewal
func (e *EmailService) SendRenewalReminder(subscription *models.Subscription, daysUntilRenewal int) error {
	// Get currency symbol
	currencySymbol := e.preferences.GetCurrencySymbol()

	// Build email body
	tmpl := `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.reminder { background-color: #d1ecf1; border: 1px solid #0c5460; border-radius: 5px; padding: 15px; margin: 20px 0; }
		.subscription-details { background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 20px 0; }
		.detail-row { margin: 10px 0; }
		.label { font-weight: bold; }
		.footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h2>{{.Title}}</h2>
		<div class="reminder">
			<strong>` + "\U0001f514" + ` {{.ReminderLabel}}</strong> {{.ReminderText}}
		</div>
		<div class="subscription-details">
			<h3>{{.DetailsTitle}}</h3>
			<div class="detail-row"><span class="label">{{.LabelName}}</span> {{.Subscription.Name}}</div>
			<div class="detail-row"><span class="label">{{.LabelCost}}</span> {{.CurrencySymbol}}{{printf "%.2f" .Subscription.Cost}} {{.Subscription.Schedule}}</div>
			<div class="detail-row"><span class="label">{{.LabelMonthlyCost}}</span> {{.CurrencySymbol}}{{printf "%.2f" (.Subscription.MonthlyCost)}}</div>
			{{if and .Subscription.Category .Subscription.Category.Name}}<div class="detail-row"><span class="label">{{.LabelCategory}}</span> {{.Subscription.Category.Name}}</div>{{end}}
			{{if .Subscription.RenewalDate}}<div class="detail-row"><span class="label">{{.LabelRenewalDate}}</span> {{.Subscription.RenewalDate.Format "January 2, 2006"}}</div>{{end}}
			{{if .Subscription.URL}}<div class="detail-row"><span class="label">{{.LabelURL}}</span> <a href="{{.Subscription.URL}}">{{.Subscription.URL}}</a></div>{{end}}
		</div>
		<div class="footer">
			<p>{{.FooterAuto}}</p>
			<p>{{.FooterManage}}</p>
		</div>
	</div>
</body>
</html>
`

	reminderText := e.tPlural("email_renewal_reminder", daysUntilRenewal, map[string]interface{}{"Name": subscription.Name})

	type ReminderData struct {
		Subscription     *models.Subscription
		DaysUntilRenewal int
		CurrencySymbol   string
		Title            string
		ReminderLabel    string
		ReminderText     string
		DetailsTitle     string
		LabelName        string
		LabelCost        string
		LabelMonthlyCost string
		LabelCategory    string
		LabelRenewalDate string
		LabelURL         string
		FooterAuto       string
		FooterManage     string
	}

	data := ReminderData{
		Subscription:     subscription,
		DaysUntilRenewal: daysUntilRenewal,
		CurrencySymbol:   currencySymbol,
		Title:            e.t("email_renewal_title"),
		ReminderLabel:    "Reminder:",
		ReminderText:     reminderText,
		DetailsTitle:     e.t("email_sub_details"),
		LabelName:        e.t("email_name"),
		LabelCost:        e.t("email_cost"),
		LabelMonthlyCost: e.t("email_monthly_cost"),
		LabelCategory:    e.t("email_category"),
		LabelRenewalDate: e.t("email_renewal_date"),
		LabelURL:         e.t("email_url"),
		FooterAuto:       e.t("email_footer_auto"),
		FooterManage:     e.t("email_footer_manage"),
	}

	tpl, err := template.New("renewalReminder").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	subject := fmt.Sprintf("%s: %s", e.t("shoutrrr_renewal_reminder"), reminderText)
	return e.SendEmail(subject, buf.String())
}

// SendCancellationReminder sends an email reminder for an upcoming subscription cancellation
func (e *EmailService) SendCancellationReminder(subscription *models.Subscription, daysUntilCancellation int) error {
	// Get currency symbol
	currencySymbol := e.preferences.GetCurrencySymbol()

	// Build email body
	tmpl := `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.reminder { background-color: #fff3cd; border: 1px solid #856404; border-radius: 5px; padding: 15px; margin: 20px 0; }
		.subscription-details { background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 20px 0; }
		.detail-row { margin: 10px 0; }
		.label { font-weight: bold; }
		.footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h2>{{.Title}}</h2>
		<div class="reminder">
			<strong>` + "\u26a0\ufe0f" + ` {{.ReminderLabel}}</strong> {{.ReminderText}}
		</div>
		<div class="subscription-details">
			<h3>{{.DetailsTitle}}</h3>
			<div class="detail-row"><span class="label">{{.LabelName}}</span> {{.Subscription.Name}}</div>
			<div class="detail-row"><span class="label">{{.LabelCost}}</span> {{.CurrencySymbol}}{{printf "%.2f" .Subscription.Cost}} {{.Subscription.Schedule}}</div>
			<div class="detail-row"><span class="label">{{.LabelMonthlyCost}}</span> {{.CurrencySymbol}}{{printf "%.2f" (.Subscription.MonthlyCost)}}</div>
			{{if and .Subscription.Category .Subscription.Category.Name}}<div class="detail-row"><span class="label">{{.LabelCategory}}</span> {{.Subscription.Category.Name}}</div>{{end}}
			{{if .Subscription.CancellationDate}}<div class="detail-row"><span class="label">{{.LabelCancellationDate}}</span> {{.Subscription.CancellationDate.Format "January 2, 2006"}}</div>{{end}}
			{{if .Subscription.URL}}<div class="detail-row"><span class="label">{{.LabelURL}}</span> <a href="{{.Subscription.URL}}">{{.Subscription.URL}}</a></div>{{end}}
		</div>
		<div class="footer">
			<p>{{.FooterAuto}}</p>
			<p>{{.FooterManage}}</p>
		</div>
	</div>
</body>
</html>
`

	reminderText := e.tPlural("email_cancellation_reminder", daysUntilCancellation, map[string]interface{}{"Name": subscription.Name})

	type CancellationReminderData struct {
		Subscription          *models.Subscription
		DaysUntilCancellation int
		CurrencySymbol        string
		Title                 string
		ReminderLabel         string
		ReminderText          string
		DetailsTitle          string
		LabelName             string
		LabelCost             string
		LabelMonthlyCost      string
		LabelCategory         string
		LabelCancellationDate string
		LabelURL              string
		FooterAuto            string
		FooterManage          string
	}

	data := CancellationReminderData{
		Subscription:          subscription,
		DaysUntilCancellation: daysUntilCancellation,
		CurrencySymbol:        currencySymbol,
		Title:                 e.t("email_cancellation_title"),
		ReminderLabel:         "Reminder:",
		ReminderText:          reminderText,
		DetailsTitle:          e.t("email_sub_details"),
		LabelName:             e.t("email_name"),
		LabelCost:             e.t("email_cost"),
		LabelMonthlyCost:      e.t("email_monthly_cost"),
		LabelCategory:         e.t("email_category"),
		LabelCancellationDate: e.t("email_cancellation_date"),
		LabelURL:              e.t("email_url"),
		FooterAuto:            e.t("email_footer_auto"),
		FooterManage:          e.t("email_footer_manage"),
	}

	tpl, err := template.New("cancellationReminder").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	subject := fmt.Sprintf("%s: %s", e.t("shoutrrr_cancellation_reminder"), reminderText)
	return e.SendEmail(subject, buf.String())
}

func (e *EmailService) SendBudgetExceededAlert(totalSpend, budget float64, currencySymbol string) error {
	config, err := e.notifConfig.GetSMTPConfig()
	if err != nil || config == nil || config.Host == "" {
		return nil
	}

	subject := e.t("email_budget_exceeded_subject")

	body := fmt.Sprintf(`<html><body style="font-family: Arial, sans-serif; padding: 20px;">
<h2>%s</h2>
<p>%s</p>
<p><strong>%s:</strong> %s%.2f</p>
<p><strong>%s:</strong> %s%.2f</p>
<p style="color: #dc2626;">%s: %s%.2f</p>
</body></html>`,
		e.t("email_budget_exceeded_subject"),
		e.t("budget_exceeded_alert"),
		e.t("dashboard_budget"), currencySymbol, budget,
		e.t("analytics_monthly_cost"), currencySymbol, totalSpend,
		e.t("dashboard_budget_exceeded"), currencySymbol, totalSpend-budget,
	)

	return e.SendEmail(subject, body)
}
