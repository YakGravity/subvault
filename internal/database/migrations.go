package database

import (
	"log/slog"
	"strconv"
	"subtrackr/internal/models"

	"gorm.io/gorm"
)

// RunMigrations executes all database migrations
func RunMigrations(db *gorm.DB) error {
	// Auto-migrate non-problematic models first
	err := db.AutoMigrate(&models.Category{}, &models.Settings{}, &models.APIKey{}, &models.ExchangeRate{})
	if err != nil {
		return err
	}

	// Run specific migrations
	migrations := []func(*gorm.DB) error{
		migrateCategoriesToDynamic,
		migrateCurrencyFields,
		migrateDateCalculationVersioning,
		migrateSubscriptionIcons,
		migrateReminderTracking,
		migrateCancellationReminderTracking,
		migrateDefaultCategory,
		migrateTaxFields,
		migrateContractFields,
		migratePerSubscriptionNotifications,
	}

	for _, migration := range migrations {
		if err := migration(db); err != nil {
			return err
		}
	}

	// Try to auto-migrate subscriptions after the category migration
	// This might fail on existing databases but that's okay
	db.AutoMigrate(&models.Subscription{})

	return nil
}

// migrateCategoriesToDynamic handles the v0.3.0 migration from string categories to category IDs
func migrateCategoriesToDynamic(db *gorm.DB) error {
	// Check if migration is needed by looking for the old category column
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='category'").Scan(&count)

	if count == 0 {
		// Migration already completed
		return nil
	}

	slog.Info("running migration: converting categories to dynamic system")

	// First ensure default categories exist
	defaultCategories := []string{"Entertainment", "Productivity", "Storage", "Software", "Fitness", "Education", "Food", "Travel", "Business", "Other"}
	var categories []models.Category
	db.Find(&categories)

	if len(categories) == 0 {
		for _, name := range defaultCategories {
			db.Create(&models.Category{Name: name})
		}
		db.Find(&categories) // Reload categories
	}

	// Create category map
	categoryMap := make(map[string]uint)
	for _, cat := range categories {
		categoryMap[cat.Name] = cat.ID
	}

	// Get all subscriptions that need migration
	type OldSubscription struct {
		ID       uint
		Category string
	}

	var oldSubs []OldSubscription
	db.Table("subscriptions").Select("id, category").Scan(&oldSubs)

	// Update each subscription with the appropriate category_id
	for _, sub := range oldSubs {
		if sub.Category != "" {
			if catID, exists := categoryMap[sub.Category]; exists {
				db.Table("subscriptions").Where("id = ?", sub.ID).Update("category_id", catID)
			} else {
				// If category doesn't exist, use "Other"
				if otherID, exists := categoryMap["Other"]; exists {
					db.Table("subscriptions").Where("id = ?", sub.ID).Update("category_id", otherID)
				}
			}
		}
	}

	// SQLite limitation: we can't drop the old category column
	// The repository layer now handles both old and new schemas transparently
	// This ensures backward compatibility without data loss

	slog.Info("migration completed: categories converted to dynamic system")
	return nil
}

// migrateCurrencyFields adds original_currency field to existing subscriptions
func migrateCurrencyFields(db *gorm.DB) error {
	// Check if original_currency column already exists
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='original_currency'").Scan(&count)

	if count > 0 {
		// Migration already completed
		return nil
	}

	slog.Info("running migration: adding currency fields")

	// Add original_currency column with default 'USD'
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN original_currency TEXT DEFAULT 'USD'").Error; err != nil {
		// Column might already exist, that's okay
		slog.Info("could not add original_currency column", "error", err)
	}

	// Set USD as default for existing subscriptions
	if err := db.Exec("UPDATE subscriptions SET original_currency = 'USD' WHERE original_currency IS NULL OR original_currency = ''").Error; err != nil {
		slog.Warn("could not update existing subscriptions with default currency", "error", err)
	}

	slog.Info("migration completed: currency fields added")
	return nil
}

// migrateDateCalculationVersioning adds date_calculation_version field for versioned date logic
func migrateDateCalculationVersioning(db *gorm.DB) error {
	// Check if date_calculation_version column already exists
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='date_calculation_version'").Scan(&count)

	if count > 0 {
		// Migration already completed
		return nil
	}

	slog.Info("running migration: adding date calculation versioning")

	// Add date_calculation_version column with default 1 (existing logic)
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN date_calculation_version INTEGER DEFAULT 1").Error; err != nil {
		// Column might already exist, that's okay
		slog.Info("could not add date_calculation_version column", "error", err)
	}

	// Set version 1 for all existing subscriptions (maintain backward compatibility)
	if err := db.Exec("UPDATE subscriptions SET date_calculation_version = 1 WHERE date_calculation_version IS NULL").Error; err != nil {
		slog.Warn("could not update existing subscriptions with default version", "error", err)
	}

	slog.Info("migration completed: date calculation versioning added")
	return nil
}

// migrateSubscriptionIcons adds icon_url field to subscriptions table
func migrateSubscriptionIcons(db *gorm.DB) error {
	// Check if icon_url column already exists
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='icon_url'").Scan(&count)

	if count > 0 {
		// Migration already completed
		return nil
	}

	slog.Info("running migration: adding subscription icon URLs")

	// Add icon_url column (nullable, empty string default)
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN icon_url TEXT DEFAULT ''").Error; err != nil {
		// Column might already exist, that's okay
		slog.Info("could not add icon_url column", "error", err)
	}

	// Set empty string as default for existing subscriptions
	if err := db.Exec("UPDATE subscriptions SET icon_url = '' WHERE icon_url IS NULL").Error; err != nil {
		slog.Warn("could not update existing subscriptions with default icon_url", "error", err)
	}

	slog.Info("migration completed: subscription icon URLs added")
	return nil
}

// migrateReminderTracking adds fields to track when reminders were sent
func migrateReminderTracking(db *gorm.DB) error {
	// Check if last_reminder_sent column already exists
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='last_reminder_sent'").Scan(&count)

	if count > 0 {
		// Migration already completed
		return nil
	}

	slog.Info("running migration: adding reminder tracking fields")

	// Add last_reminder_sent column
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN last_reminder_sent DATETIME").Error; err != nil {
		slog.Info("could not add last_reminder_sent column", "error", err)
	}

	// Add last_reminder_renewal_date column
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN last_reminder_renewal_date DATETIME").Error; err != nil {
		slog.Info("could not add last_reminder_renewal_date column", "error", err)
	}

	slog.Info("migration completed: reminder tracking fields added")
	return nil
}

// migrateCancellationReminderTracking adds fields to track when cancellation reminders were sent
func migrateCancellationReminderTracking(db *gorm.DB) error {
	// Check if last_cancellation_reminder_sent column already exists
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='last_cancellation_reminder_sent'").Scan(&count)

	if count > 0 {
		// Migration already completed
		return nil
	}

	slog.Info("running migration: adding cancellation reminder tracking fields")

	// Add last_cancellation_reminder_sent column
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN last_cancellation_reminder_sent DATETIME").Error; err != nil {
		slog.Info("could not add last_cancellation_reminder_sent column", "error", err)
	}

	// Add last_cancellation_reminder_date column
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN last_cancellation_reminder_date DATETIME").Error; err != nil {
		slog.Info("could not add last_cancellation_reminder_date column", "error", err)
	}

	slog.Info("migration completed: cancellation reminder tracking fields added")
	return nil
}

// migrateDefaultCategory adds is_default column and creates a default category
func migrateDefaultCategory(db *gorm.DB) error {
	// Add is_default column if not exists
	if !db.Migrator().HasColumn(&models.Category{}, "is_default") {
		db.Migrator().AddColumn(&models.Category{}, "IsDefault")
	}

	// Create default category if none exists
	var count int64
	db.Model(&models.Category{}).Where("is_default = ?", true).Count(&count)
	if count == 0 {
		db.Create(&models.Category{
			Name:      "General",
			IsDefault: true,
		})
	}
	return nil
}

// migrateTaxFields adds tax_rate and price_type columns to subscriptions
func migrateTaxFields(db *gorm.DB) error {
	if !db.Migrator().HasColumn(&models.Subscription{}, "tax_rate") {
		db.Migrator().AddColumn(&models.Subscription{}, "TaxRate")
	}
	if !db.Migrator().HasColumn(&models.Subscription{}, "price_type") {
		db.Migrator().AddColumn(&models.Subscription{}, "PriceType")
	}
	return nil
}

// migrateContractFields adds customer_number, contract_number, and login_name columns to subscriptions
func migrateContractFields(db *gorm.DB) error {
	if !db.Migrator().HasColumn(&models.Subscription{}, "customer_number") {
		db.Migrator().AddColumn(&models.Subscription{}, "CustomerNumber")
	}
	if !db.Migrator().HasColumn(&models.Subscription{}, "contract_number") {
		db.Migrator().AddColumn(&models.Subscription{}, "ContractNumber")
	}
	if !db.Migrator().HasColumn(&models.Subscription{}, "login_name") {
		db.Migrator().AddColumn(&models.Subscription{}, "LoginName")
	}
	// Migrate data from account to login_name
	db.Exec("UPDATE subscriptions SET login_name = account WHERE account != '' AND account IS NOT NULL AND (login_name = '' OR login_name IS NULL)")
	return nil
}

func migratePerSubscriptionNotifications(db *gorm.DB) error {
	columns := map[string]string{
		"renewal_reminder":           "RenewalReminder",
		"renewal_reminder_days":      "RenewalReminderDays",
		"cancellation_reminder":      "CancellationReminder",
		"cancellation_reminder_days": "CancellationReminderDays",
		"high_cost_alert":            "HighCostAlert",
	}
	for col, field := range columns {
		if !db.Migrator().HasColumn(&models.Subscription{}, col) {
			db.Migrator().AddColumn(&models.Subscription{}, field)
		}
	}

	// Migrate global settings values to existing subscriptions
	type settingRow struct {
		Value string
	}
	getSettingValue := func(key string) string {
		var row settingRow
		if err := db.Table("settings").Select("value").Where("`key` = ?", key).First(&row).Error; err != nil {
			return ""
		}
		return row.Value
	}

	renewalEnabled := getSettingValue("renewal_reminders") == "true"
	cancellationEnabled := getSettingValue("cancellation_reminders") == "true"
	highCostEnabled := getSettingValue("high_cost_alerts")
	// high_cost_alerts defaults to true if not set
	highCostAlertOn := highCostEnabled == "" || highCostEnabled == "true"

	reminderDaysStr := getSettingValue("reminder_days")
	reminderDays := 3
	if reminderDaysStr != "" {
		if v, err := strconv.Atoi(reminderDaysStr); err == nil {
			reminderDays = v
		}
	}

	cancellationDaysStr := getSettingValue("cancellation_reminder_days")
	cancellationDays := 7
	if cancellationDaysStr != "" {
		if v, err := strconv.Atoi(cancellationDaysStr); err == nil {
			cancellationDays = v
		}
	}

	// Apply global settings to all existing subscriptions that still have defaults
	db.Exec("UPDATE subscriptions SET renewal_reminder = ?, renewal_reminder_days = ? WHERE renewal_reminder = 0 OR renewal_reminder IS NULL",
		renewalEnabled, reminderDays)
	db.Exec("UPDATE subscriptions SET cancellation_reminder = ?, cancellation_reminder_days = ? WHERE cancellation_reminder = 0 OR cancellation_reminder IS NULL",
		cancellationEnabled, cancellationDays)
	db.Exec("UPDATE subscriptions SET high_cost_alert = ? WHERE high_cost_alert = 0 OR high_cost_alert IS NULL",
		highCostAlertOn)

	return nil
}
