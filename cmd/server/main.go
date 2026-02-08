package main

import (
	"crypto/subtle"
	"flag"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strings"
	"subvault/internal/config"
	"subvault/internal/database"
	"subvault/internal/handlers"
	"subvault/internal/i18n"
	"subvault/internal/middleware"
	"subvault/internal/repository"
	"subvault/internal/service"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/term"
)

func main() {
	// Setup structured logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	// CLI flags
	resetPassword := flag.Bool("reset-password", false, "Reset admin password (interactive or with --new-password)")
	newPassword := flag.String("new-password", "", "New password for admin (non-interactive, use with --reset-password)")
	disableAuth := flag.Bool("disable-auth", false, "Disable authentication and remove credentials")
	flag.Parse()

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabasePath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Run database migrations
	err = database.RunMigrations(db)
	if err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Initialize repositories
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	exchangeRateRepo := repository.NewExchangeRateRepository(db)

	// Initialize i18n service
	i18nService := i18n.NewI18nService()

	// Initialize services
	categoryService := service.NewCategoryService(categoryRepo)
	settingsService := service.NewSettingsService(settingsRepo)
	currencyService := service.NewCurrencyService(exchangeRateRepo, settingsService)
	preferencesService := service.NewPreferencesService(settingsService)
	authService := service.NewAuthService(settingsService, settingsRepo)
	apiKeyService := service.NewAPIKeyService(settingsRepo)
	notifConfigService := service.NewNotificationConfigService(settingsService, settingsRepo)
	calendarService := service.NewCalendarService(settingsService, settingsRepo)
	renewalService := service.NewRenewalService()
	subscriptionService := service.NewSubscriptionService(subscriptionRepo, categoryService, currencyService, preferencesService, settingsService, renewalService)
	emailService := service.NewEmailService(preferencesService, notifConfigService, i18nService)
	shoutrrrService := service.NewShoutrrrService(preferencesService, notifConfigService, i18nService)

	// Migrate existing Pushover config to Shoutrrr format (one-time migration)
	if err := notifConfigService.MigratePushoverToShoutrrr(); err != nil {
		slog.Warn("pushover to shoutrrr migration failed", "error", err)
	}
	logoService := service.NewLogoService()

	// Handle CLI commands (run before starting HTTP server)
	if *disableAuth {
		handleDisableAuth(authService)
		return
	}

	if *resetPassword {
		handleResetPassword(authService, *newPassword)
		return
	}

	// Initialize session service (get or generate session secret)
	sessionSecret, err := authService.GetOrGenerateSessionSecret()
	if err != nil {
		log.Fatal("Failed to initialize session secret:", err)
	}
	sessionService := service.NewSessionService(sessionSecret)

	// Initialize CSRF secret
	csrfSecret, err := authService.GetOrGenerateCSRFSecret()
	if err != nil {
		log.Fatal("Failed to initialize CSRF secret:", err)
	}

	// Initialize handlers
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionService, preferencesService, settingsService, calendarService, currencyService, emailService, shoutrrrService, logoService)
	settingsHandler := handlers.NewSettingsHandler(settingsService, authService, apiKeyService, preferencesService, notifConfigService, calendarService, currencyService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	authHandler := handlers.NewAuthHandler(authService, sessionService, emailService, notifConfigService)
	importHandler := handlers.NewImportHandler(subscriptionService, categoryService, settingsService)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Load HTML templates with error handling
	tmpl := loadTemplates()
	if tmpl != nil && len(tmpl.Templates()) > 0 {
		router.SetHTMLTemplate(tmpl)
	} else {
		slog.Warn("template loading failed, using fallback")
		// Fallback to LoadHTMLGlob for compatibility
		router.LoadHTMLGlob("templates/*")
	}

	// Serve static files with cache headers
	staticFS := http.Dir("./web/static")
	staticHandler := http.StripPrefix("/static/", http.FileServer(staticFS))
	router.GET("/static/*filepath", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=86400")
		staticHandler.ServeHTTP(c.Writer, c.Request)
	})
	router.HEAD("/static/*filepath", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=86400")
		staticHandler.ServeHTTP(c.Writer, c.Request)
	})
	router.StaticFile("/favicon.ico", "./web/static/favicon.ico")
	router.StaticFile("/manifest.json", "./web/static/manifest.json")

	// Health check endpoint with database connectivity check
	router.GET("/healthz", func(c *gin.Context) {
		// Check database connectivity
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "database connection unavailable",
			})
			return
		}

		// Ping the database to verify connectivity
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "database ping failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// Apply CSRF middleware (before auth - login page needs CSRF too)
	csrfSecure := os.Getenv("HTTPS_ENABLED") == "true"
	router.Use(middleware.CSRFMiddleware(csrfSecret, csrfSecure))

	// Apply auth middleware
	router.Use(middleware.AuthMiddleware(authService, sessionService))

	// Apply i18n middleware
	router.Use(middleware.I18nMiddleware(i18nService, preferencesService))

	// Routes
	setupRoutes(router, subscriptionHandler, settingsHandler, apiKeyService, categoryHandler, authHandler, importHandler)

	// Seed sample data if database is empty
	// Commented out - no sample data by default
	// if subscriptionService.Count() == 0 {
	// 	seedSampleData(subscriptionService)
	// }

	// Start renewal reminder scheduler
	go startRenewalReminderScheduler(subscriptionService, emailService, shoutrrrService, settingsService)

	// Start cancellation reminder scheduler
	go startCancellationReminderScheduler(subscriptionService, emailService, shoutrrrService, settingsService)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("server starting", "port", port)
	log.Fatal(router.Run(":" + port))
}

// loadTemplates loads HTML templates with better error handling for arm64 compatibility
func loadTemplates() *template.Template {
	tmpl := template.New("")

	// Add template functions
	tmpl.Funcs(template.FuncMap{
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			m := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				m[key] = values[i+1]
			}
			return m
		},
		"hasPrefix": strings.HasPrefix,
		"add":       func(a, b float64) float64 { return a + b },
		"sub":       func(a, b float64) float64 { return a - b },
		"mul":       func(a, b float64) float64 { return a * b },
		"div": func(a, b float64) float64 {
			if b == 0 {
				slog.Warn("division by zero attempted in template")
				return math.NaN()
			}
			return a / b
		},
		"int": func(v interface{}) int {
			switch val := v.(type) {
			case int:
				return val
			case int64:
				return int(val)
			case float64:
				return int(val)
			case time.Month:
				return int(val)
			default:
				return 0
			}
		},
	})

	// Load partials first (they define reusable template blocks)
	partialFiles := []string{
		"templates/partials/sidebar.html",
	}
	for _, file := range partialFiles {
		if _, err := tmpl.ParseFiles(file); err != nil {
			slog.Error("failed to parse partial", "file", file, "error", err)
		}
	}

	// Critical templates required for basic functionality
	criticalTemplates := []string{
		"templates/dashboard.html",
		"templates/subscriptions.html",
		"templates/error.html",
	}

	// All template files to load
	templateFiles := []string{
		"templates/dashboard.html",
		"templates/subscriptions.html",
		"templates/calendar.html",
		"templates/settings-general.html",
		"templates/settings-notifications.html",
		"templates/settings-data.html",
		"templates/settings-security.html",
		"templates/settings-appearance.html",
		"templates/api-docs.html",
		"templates/subscription-form.html",
		"templates/subscription-list.html",
		"templates/categories-list.html",
		"templates/api-keys-list.html",
		"templates/smtp-message.html",
		"templates/form-errors.html",
		"templates/error.html",
		"templates/login.html",
		"templates/login-error.html",
		"templates/forgot-password.html",
		"templates/forgot-password-error.html",
		"templates/forgot-password-success.html",
		"templates/reset-password.html",
		"templates/reset-password-error.html",
		"templates/reset-password-success.html",
		"templates/auth-message.html",
		"templates/import-result.html",
		"templates/exchange-rate-status.html",
	}

	var parsedCount int
	var failedCount int
	var missingCritical []string

	// Load templates individually to catch arm64-specific issues
	for _, file := range templateFiles {
		if _, err := os.Stat(file); err != nil {
			slog.Warn("template file not found", "file", file)
			// Check if this is a critical template
			for _, critical := range criticalTemplates {
				if critical == file {
					missingCritical = append(missingCritical, file)
				}
			}
			continue
		}

		if _, err := tmpl.ParseFiles(file); err != nil {
			slog.Error("failed to parse template", "file", file, "error", err)
			failedCount++
			// Check if this is a critical template
			for _, critical := range criticalTemplates {
				if critical == file {
					missingCritical = append(missingCritical, file)
				}
			}
		} else {
			parsedCount++
		}
	}

	// Log template loading summary
	slog.Info("template loading complete", "parsed", parsedCount, "failed", failedCount, "total", len(templateFiles))

	// Fatal error if critical templates are missing
	if len(missingCritical) > 0 {
		log.Fatalf("Critical templates failed to load: %v. Application cannot continue.", missingCritical)
	}

	// Warn if too many templates failed
	if failedCount > len(templateFiles)/2 {
		slog.Warn("more than half of templates failed to load", "failed", failedCount, "total", len(templateFiles))
	}

	return tmpl
}

func setupRoutes(router *gin.Engine, handler *handlers.SubscriptionHandler, settingsHandler *handlers.SettingsHandler, apiKeyService *service.APIKeyService, categoryHandler *handlers.CategoryHandler, authHandler *handlers.AuthHandler, importHandler *handlers.ImportHandler) {
	// Calendar feed (public, token-based auth)
	router.GET("/cal/:token/subscriptions.ics", handler.ServeCalendarFeed)

	// Auth routes (public)
	router.GET("/login", authHandler.ShowLoginPage)
	router.GET("/forgot-password", authHandler.ShowForgotPasswordPage)
	router.GET("/reset-password", authHandler.ShowResetPasswordPage)

	// Web routes
	router.GET("/", handler.Dashboard)
	router.GET("/dashboard", handler.Dashboard)
	router.GET("/subscriptions", handler.SubscriptionsList)
	router.GET("/analytics", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/dashboard")
	})
	router.GET("/calendar", handler.Calendar)
	router.GET("/settings", settingsHandler.SettingsGeneral)
	router.GET("/settings/notifications", settingsHandler.SettingsNotifications)
	router.GET("/settings/data", settingsHandler.SettingsData)
	router.GET("/settings/security", settingsHandler.SettingsSecurity)
	router.GET("/settings/appearance", settingsHandler.SettingsAppearance)
	router.GET("/api-docs", settingsHandler.APIDocs)

	// Form routes for HTMX modals
	form := router.Group("/form")
	{
		form.GET("/subscription", handler.GetSubscriptionForm)
		form.GET("/subscription/:id", handler.GetSubscriptionForm)
	}

	// API routes for HTMX
	api := router.Group("/api")
	{
		api.GET("/subscriptions", handler.GetSubscriptions)
		api.POST("/subscriptions", handler.CreateSubscription)
		api.GET("/subscriptions/:id", handler.GetSubscription)
		api.PUT("/subscriptions/:id", handler.UpdateSubscription)
		api.DELETE("/subscriptions/:id", handler.DeleteSubscription)
		api.GET("/stats", handler.GetStats)

		// Export and data management routes
		api.GET("/export/csv", handler.ExportCSV)
		api.GET("/export/json", handler.ExportJSON)
		api.GET("/export/ical", handler.ExportICal)
		api.GET("/backup", handler.BackupData)
		api.DELETE("/clear-all", handler.ClearAllData)

		// Calendar token management
		api.POST("/calendar/generate", settingsHandler.GenerateCalendarToken)
		api.POST("/calendar/revoke", settingsHandler.RevokeCalendarToken)

		// Settings routes
		api.POST("/settings/smtp", settingsHandler.SaveSMTPSettings)
		api.POST("/settings/smtp/test", settingsHandler.TestSMTPConnection)
		api.POST("/settings/shoutrrr", settingsHandler.SaveShoutrrrSettings)
		api.POST("/settings/shoutrrr/test", settingsHandler.TestShoutrrrConnection)
		api.GET("/settings/shoutrrr", settingsHandler.GetShoutrrrConfig)
		api.POST("/settings/notifications/:setting", settingsHandler.UpdateNotificationSetting)
		api.GET("/settings/notifications", settingsHandler.GetNotificationSettings)
		api.GET("/settings/smtp", settingsHandler.GetSMTPConfig)

		// API Key management routes
		api.GET("/settings/apikeys", settingsHandler.ListAPIKeys)
		api.POST("/settings/apikeys", settingsHandler.CreateAPIKey)
		api.DELETE("/settings/apikeys/:id", settingsHandler.DeleteAPIKey)

		// Currency setting
		api.POST("/settings/currency", settingsHandler.UpdateCurrency)

		// Exchange rate management
		api.POST("/settings/exchange-rates/refresh", settingsHandler.RefreshExchangeRates)
		api.POST("/settings/currency-refresh", settingsHandler.UpdateCurrencyRefreshInterval)

		// Language setting
		api.POST("/settings/language", settingsHandler.UpdateLanguage)

		// Dark mode setting
		api.POST("/settings/dark-mode", settingsHandler.ToggleDarkMode)

		// Import routes
		api.POST("/import/subscriptions", importHandler.ImportSubscriptions)
		api.POST("/import/encrypted", importHandler.ImportEncrypted)

		// Encrypted export route
		api.POST("/export/encrypted", handler.ExportEncrypted)

		// Category management routes
		api.GET("/categories", categoryHandler.ListCategories)
		api.POST("/categories", categoryHandler.CreateCategory)
		api.PUT("/categories/:id", categoryHandler.UpdateCategory)
		api.DELETE("/categories/:id", categoryHandler.DeleteCategory)

		// Auth routes
		api.POST("/auth/login", authHandler.Login)
		api.GET("/auth/logout", authHandler.Logout)
		api.POST("/auth/forgot-password", authHandler.ForgotPassword)
		api.POST("/auth/reset-password", authHandler.ResetPassword)

		// Auth settings routes
		api.POST("/settings/auth/setup", settingsHandler.SetupAuth)
		api.POST("/settings/auth/disable", settingsHandler.DisableAuth)
		api.GET("/settings/auth/status", settingsHandler.GetAuthStatus)

		// Theme settings routes
		api.GET("/settings/theme", settingsHandler.GetTheme)
		api.POST("/settings/theme", settingsHandler.SetTheme)

		// Date format settings routes
		api.GET("/settings/date-format", settingsHandler.GetDateFormat)
		api.POST("/settings/date-format", settingsHandler.SetDateFormat)
	}

	// Public API routes (require API key authentication)
	v1 := router.Group("/api/v1")
	v1.Use(middleware.APIKeyAuth(apiKeyService))
	{
		// Subscription endpoints
		v1.GET("/subscriptions", handler.GetSubscriptionsAPI)
		v1.POST("/subscriptions", handler.CreateSubscriptionAPI)
		v1.GET("/subscriptions/:id", handler.GetSubscription)
		v1.PUT("/subscriptions/:id", handler.UpdateSubscriptionAPI)
		v1.DELETE("/subscriptions/:id", handler.DeleteSubscriptionAPI)

		// Stats and export endpoints
		v1.GET("/stats", handler.GetStats)
		v1.GET("/export/csv", handler.ExportCSV)
		v1.GET("/export/json", handler.ExportJSON)
		v1.GET("/export/ical", handler.ExportICal)

		// Category endpoints
		v1.GET("/categories", categoryHandler.ListCategories)
		v1.POST("/categories", categoryHandler.CreateCategory)
		v1.PUT("/categories/:id", categoryHandler.UpdateCategory)
		v1.DELETE("/categories/:id", categoryHandler.DeleteCategory)
	}
}

// startRenewalReminderScheduler starts a background goroutine that checks for
// upcoming renewals and sends reminder emails and Shoutrrr notifications daily
func startRenewalReminderScheduler(subscriptionService *service.SubscriptionService, emailService *service.EmailService, shoutrrrService *service.ShoutrrrService, settingsService *service.SettingsService) {
	// Run immediately on startup (after a short delay to let server initialize)
	go func() {
		time.Sleep(30 * time.Second) // Wait 30 seconds for server to fully start
		checkAndSendRenewalReminders(subscriptionService, emailService, shoutrrrService, settingsService)
	}()

	// Then run daily at midnight
	// Note: Ticker is intentionally not stopped as this is a long-running server process.
	// The ticker will run for the lifetime of the application, which is the desired behavior.
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		defer ticker.Stop() // Clean up ticker if goroutine exits (defensive programming)
		for range ticker.C {
			// Recover from any panics in the reminder check to keep the scheduler running
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("panic in renewal reminder check", "panic", r)
					}
				}()
				checkAndSendRenewalReminders(subscriptionService, emailService, shoutrrrService, settingsService)
			}()
		}
	}()
}

// checkAndSendRenewalReminders checks for subscriptions needing reminders and sends emails and Shoutrrr notifications
func checkAndSendRenewalReminders(subscriptionService *service.SubscriptionService, emailService *service.EmailService, shoutrrrService *service.ShoutrrrService, settingsService *service.SettingsService) {
	// Get subscriptions needing reminders (per-subscription settings)
	subscriptions, err := subscriptionService.GetSubscriptionsNeedingReminders()
	if err != nil {
		slog.Error("failed to get subscriptions for renewal reminders", "error", err)
		return
	}

	if len(subscriptions) == 0 {
		slog.Info("no subscriptions need renewal reminders today")
		return
	}

	slog.Info("checking subscriptions for renewal reminders", "count", len(subscriptions))

	// Send reminder for each subscription (both email and Shoutrrr)
	sentCount := 0
	failedCount := 0
	for sub, daysUntil := range subscriptions {
		emailErr := emailService.SendRenewalReminder(sub, daysUntil)
		shoutrrrErr := shoutrrrService.SendRenewalReminder(sub, daysUntil)

		// If both fail, count as failed; otherwise consider it sent
		if emailErr != nil && shoutrrrErr != nil {
			slog.Error("failed to send renewal reminder", "subscription", sub.Name, "id", sub.ID, "emailError", emailErr, "shoutrrrError", shoutrrrErr)
			failedCount++
		} else {
			// Mark reminder as sent for this renewal date
			now := time.Now()
			sub.LastReminderSent = &now
			if sub.RenewalDate != nil {
				renewalDateCopy := *sub.RenewalDate
				sub.LastReminderRenewalDate = &renewalDateCopy
			}

			// Update the subscription in the database
			_, updateErr := subscriptionService.Update(sub.ID, sub)
			if updateErr != nil {
				slog.Warn("failed to update last reminder sent", "subscription", sub.Name, "id", sub.ID, "error", updateErr)
			}

			if emailErr != nil {
				slog.Info("sent shoutrrr renewal reminder", "subscription", sub.Name, "daysUntil", daysUntil, "emailError", emailErr)
			} else if shoutrrrErr != nil {
				slog.Info("sent email renewal reminder", "subscription", sub.Name, "daysUntil", daysUntil, "shoutrrrError", shoutrrrErr)
			} else {
				slog.Info("sent renewal reminders", "subscription", sub.Name, "daysUntil", daysUntil)
			}
			sentCount++
		}
	}

	slog.Info("renewal reminder check complete", "sent", sentCount, "failed", failedCount)
}

// startCancellationReminderScheduler starts a background goroutine that checks for
// upcoming cancellations and sends reminder emails and Shoutrrr notifications daily
func startCancellationReminderScheduler(subscriptionService *service.SubscriptionService, emailService *service.EmailService, shoutrrrService *service.ShoutrrrService, settingsService *service.SettingsService) {
	// Run immediately on startup (after a short delay to let server initialize)
	go func() {
		time.Sleep(30 * time.Second) // Wait 30 seconds for server to fully start
		checkAndSendCancellationReminders(subscriptionService, emailService, shoutrrrService, settingsService)
	}()

	// Then run daily at midnight
	// Note: Ticker is intentionally not stopped as this is a long-running server process.
	// The ticker will run for the lifetime of the application, which is the desired behavior.
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		defer ticker.Stop() // Clean up ticker if goroutine exits (defensive programming)
		for range ticker.C {
			// Recover from any panics in the reminder check to keep the scheduler running
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("panic in cancellation reminder check", "panic", r)
					}
				}()
				checkAndSendCancellationReminders(subscriptionService, emailService, shoutrrrService, settingsService)
			}()
		}
	}()
}

// checkAndSendCancellationReminders checks for subscriptions needing cancellation reminders and sends emails and Shoutrrr notifications
func checkAndSendCancellationReminders(subscriptionService *service.SubscriptionService, emailService *service.EmailService, shoutrrrService *service.ShoutrrrService, settingsService *service.SettingsService) {
	// Get subscriptions needing cancellation reminders (per-subscription settings)
	subscriptions, err := subscriptionService.GetSubscriptionsNeedingCancellationReminders()
	if err != nil {
		slog.Error("failed to get subscriptions for cancellation reminders", "error", err)
		return
	}

	if len(subscriptions) == 0 {
		slog.Info("no subscriptions need cancellation reminders today")
		return
	}

	slog.Info("checking subscriptions for cancellation reminders", "count", len(subscriptions))

	// Send reminder for each subscription (both email and Shoutrrr)
	sentCount := 0
	failedCount := 0
	for sub, daysUntil := range subscriptions {
		emailErr := emailService.SendCancellationReminder(sub, daysUntil)
		shoutrrrErr := shoutrrrService.SendCancellationReminder(sub, daysUntil)

		// If both fail, count as failed; otherwise consider it sent
		if emailErr != nil && shoutrrrErr != nil {
			slog.Error("failed to send cancellation reminder", "subscription", sub.Name, "id", sub.ID, "emailError", emailErr, "shoutrrrError", shoutrrrErr)
			failedCount++
		} else {
			// Mark reminder as sent for this cancellation date
			now := time.Now()
			sub.LastCancellationReminderSent = &now
			if sub.CancellationDate != nil {
				cancellationDateCopy := *sub.CancellationDate
				sub.LastCancellationReminderDate = &cancellationDateCopy
			}

			// Update the subscription in the database
			_, updateErr := subscriptionService.Update(sub.ID, sub)
			if updateErr != nil {
				slog.Warn("failed to update last cancellation reminder sent", "subscription", sub.Name, "id", sub.ID, "error", updateErr)
			}

			if emailErr != nil {
				slog.Info("sent shoutrrr cancellation reminder", "subscription", sub.Name, "daysUntil", daysUntil, "emailError", emailErr)
			} else if shoutrrrErr != nil {
				slog.Info("sent email cancellation reminder", "subscription", sub.Name, "daysUntil", daysUntil, "shoutrrrError", shoutrrrErr)
			} else {
				slog.Info("sent cancellation reminders", "subscription", sub.Name, "daysUntil", daysUntil)
			}
			sentCount++
		}
	}

	slog.Info("cancellation reminder check complete", "sent", sentCount, "failed", failedCount)
}

// handleResetPassword handles the --reset-password CLI command
func handleResetPassword(authService *service.AuthService, newPassword string) {
	var password string

	if newPassword != "" {
		// Non-interactive mode
		password = newPassword
	} else {
		// Interactive mode - prompt for password
		fmt.Print("Enter new admin password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatal("Failed to read password:", err)
		}
		fmt.Println()

		fmt.Print("Confirm password: ")
		confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatal("Failed to read confirmation:", err)
		}
		fmt.Println()

		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare(passwordBytes, confirmBytes) != 1 {
			log.Fatal("Passwords do not match")
		}

		password = string(passwordBytes)
	}

	// Validate password length
	if len(password) < 8 {
		log.Fatal("Password must be at least 8 characters long")
	}

	// Update password
	if err := authService.SetAuthPassword(password); err != nil {
		log.Fatal("Failed to update password:", err)
	}

	fmt.Println("✓ Admin password reset successfully")
	os.Exit(0)
}

// handleDisableAuth handles the --disable-auth CLI command
func handleDisableAuth(authService *service.AuthService) {
	if err := authService.DisableAuth(); err != nil {
		log.Fatal("Failed to disable authentication:", err)
	}

	fmt.Println("✓ Authentication disabled successfully")
	fmt.Println("  Note: Credentials are preserved and can be re-enabled from Settings")
	os.Exit(0)
}
