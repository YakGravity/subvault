package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"subvault/internal/crypto"
	"subvault/internal/models"
	"subvault/internal/service"

	"github.com/gin-gonic/gin"
)

type ImportHandler struct {
	subscriptionService service.SubscriptionServiceInterface
	categoryService     service.CategoryServiceInterface
	settingsService     service.SettingsServiceInterface
}

func NewImportHandler(subscriptionService service.SubscriptionServiceInterface, categoryService service.CategoryServiceInterface, settingsService service.SettingsServiceInterface) *ImportHandler {
	return &ImportHandler{
		subscriptionService: subscriptionService,
		categoryService:     categoryService,
		settingsService:     settingsService,
	}
}

type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   int      `json:"errors"`
	Details  []string `json:"details"`
}

// wallosNameObj represents a nested Wallos object with a name field
type wallosNameObj struct {
	Name string `json:"name"`
}

// wallosSubscription represents a subscription from Wallos export
// Supports both real Wallos format (nested objects) and flat format
type wallosSubscription struct {
	Name              string          `json:"name"`
	Price             json.RawMessage `json:"price"`
	CurrencyCode      string          `json:"currency_code"`
	Currency          wallosNameObj   `json:"currency"`
	Cycle             int             `json:"cycle"`
	Frequency         int             `json:"frequency"`
	NextPayment       string          `json:"next_payment"`
	StartDate         string          `json:"start_date"`
	CategoryName      string          `json:"category_name"`
	Category          wallosNameObj   `json:"category"`
	URL               string          `json:"url"`
	Notes             string          `json:"notes"`
	PaymentMethodName string          `json:"payment_method_name"`
	PaymentMethod     wallosNameObj   `json:"payment_method"`
}

// GetPrice returns the price as a string, handling both float and string JSON values
func (ws *wallosSubscription) GetPrice() string {
	if ws.Price == nil {
		return "0"
	}
	s := strings.TrimSpace(string(ws.Price))
	// Remove quotes if it's a JSON string
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// GetCurrencyCode returns the currency code from either flat or nested format
func (ws *wallosSubscription) GetCurrencyCode() string {
	if ws.CurrencyCode != "" {
		return ws.CurrencyCode
	}
	return ws.Currency.Name
}

// GetCategoryName returns the category name from either flat or nested format
func (ws *wallosSubscription) GetCategoryName() string {
	if ws.CategoryName != "" {
		return ws.CategoryName
	}
	return ws.Category.Name
}

// GetPaymentMethodName returns the payment method from either flat or nested format
func (ws *wallosSubscription) GetPaymentMethodName() string {
	if ws.PaymentMethodName != "" {
		return ws.PaymentMethodName
	}
	return ws.PaymentMethod.Name
}

type wallosExport struct {
	Subscriptions []wallosSubscription `json:"subscriptions"`
}

// subtrackrExport represents the SubTrackr JSON export format
type subtrackrExport struct {
	Subscriptions []models.Subscription `json:"subscriptions"`
}

func (h *ImportHandler) ImportSubscriptions(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrNoFileUploaded})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrFailedReadFile})
		return
	}

	format := c.PostForm("format")
	if format == "" {
		format = h.detectFormat(data)
	}

	var result ImportResult
	switch format {
	case "wallos":
		result = h.importWallos(data)
	case "subvault", "subtrackr":
		result = h.importSubTrackr(data)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown format"})
		return
	}

	c.HTML(http.StatusOK, "import-result.html", gin.H{
		"Result": result,
	})
}

func (h *ImportHandler) detectFormat(data []byte) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return ""
	}

	// SubTrackr exports have "exported_at" and "total_count"
	if _, ok := raw["exported_at"]; ok {
		return "subtrackr"
	}

	// Check if subscriptions array contains Wallos-specific fields
	if subsData, ok := raw["subscriptions"]; ok {
		var subs []map[string]interface{}
		if err := json.Unmarshal(subsData, &subs); err == nil && len(subs) > 0 {
			if _, hasCycle := subs[0]["cycle"]; hasCycle {
				return "wallos"
			}
			if _, hasSchedule := subs[0]["schedule"]; hasSchedule {
				return "subtrackr"
			}
		}
	}

	return ""
}

func (h *ImportHandler) importWallos(data []byte) ImportResult {
	result := ImportResult{}

	var export wallosExport
	if err := json.Unmarshal(data, &export); err != nil {
		result.Errors++
		result.Details = append(result.Details, fmt.Sprintf("Parse error: %s", err.Error()))
		return result
	}

	if len(export.Subscriptions) == 0 {
		result.Details = append(result.Details, "No subscriptions found in file")
		return result
	}

	existing, _ := h.subscriptionService.GetAll()

	for _, ws := range export.Subscriptions {
		priceStr := ws.GetPrice()

		// Duplicate check
		if h.isDuplicate(existing, ws.Name, priceStr) {
			result.Skipped++
			result.Details = append(result.Details, fmt.Sprintf("Skipped (duplicate): %s", ws.Name))
			continue
		}

		sub := models.Subscription{
			Name:                   ws.Name,
			OriginalCurrency:       ws.GetCurrencyCode(),
			Status:                 "Active",
			URL:                    ws.URL,
			Notes:                  ws.Notes,
			PaymentMethod:          ws.GetPaymentMethodName(),
			DateCalculationVersion: 2,
		}

		// Parse price
		var price float64
		fmt.Sscanf(priceStr, "%f", &price)
		sub.Cost = price

		// Map cycle to schedule
		schedule := "Monthly"
		switch ws.Cycle {
		case 1:
			schedule = "Daily"
		case 2:
			schedule = "Weekly"
		case 3:
			schedule = "Monthly"
		case 4:
			schedule = "Annual"
		}
		// Handle frequency multiplier
		if ws.Frequency > 1 && ws.Cycle == 3 && ws.Frequency == 3 {
			schedule = "Quarterly"
		}
		sub.Schedule = schedule

		// Parse next_payment as renewal date
		if ws.NextPayment != "" {
			if t, err := time.Parse("2006-01-02", ws.NextPayment); err == nil {
				sub.RenewalDate = &t
			}
		}

		// Parse start_date if available
		if ws.StartDate != "" {
			if t, err := time.Parse("2006-01-02", ws.StartDate); err == nil {
				sub.StartDate = &t
			}
		}

		// Map category
		catName := ws.GetCategoryName()
		if catName != "" {
			cat := h.getOrCreateCategory(catName)
			if cat != nil {
				sub.CategoryID = cat.ID
			}
		}

		if _, err := h.subscriptionService.Create(&sub); err != nil {
			result.Errors++
			result.Details = append(result.Details, fmt.Sprintf("Error importing %s: %s", ws.Name, err.Error()))
		} else {
			result.Imported++
		}
	}

	return result
}

func (h *ImportHandler) importSubTrackr(data []byte) ImportResult {
	result := ImportResult{}

	var export subtrackrExport
	if err := json.Unmarshal(data, &export); err != nil {
		result.Errors++
		result.Details = append(result.Details, fmt.Sprintf("Parse error: %s", err.Error()))
		return result
	}

	if len(export.Subscriptions) == 0 {
		result.Details = append(result.Details, "No subscriptions found in file")
		return result
	}

	existing, _ := h.subscriptionService.GetAll()

	for _, sub := range export.Subscriptions {
		// Duplicate check
		priceStr := fmt.Sprintf("%.2f", sub.Cost)
		if h.isDuplicate(existing, sub.Name, priceStr) {
			result.Skipped++
			result.Details = append(result.Details, fmt.Sprintf("Skipped (duplicate): %s", sub.Name))
			continue
		}

		// Reset ID and timestamps for re-import
		newSub := sub
		newSub.ID = 0
		newSub.Category = models.Category{}
		newSub.CategoryID = 0
		newSub.CreatedAt = time.Time{}
		newSub.UpdatedAt = time.Time{}
		newSub.LastReminderSent = nil
		newSub.LastReminderRenewalDate = nil
		newSub.LastCancellationReminderSent = nil
		newSub.LastCancellationReminderDate = nil

		// Map category by name if possible
		if sub.Category.Name != "" {
			cat := h.getOrCreateCategory(sub.Category.Name)
			if cat != nil {
				newSub.CategoryID = cat.ID
			}
		}

		if _, err := h.subscriptionService.Create(&newSub); err != nil {
			result.Errors++
			result.Details = append(result.Details, fmt.Sprintf("Error importing %s: %s", sub.Name, err.Error()))
		} else {
			result.Imported++
		}
	}

	return result
}

func (h *ImportHandler) isDuplicate(existing []models.Subscription, name string, price string) bool {
	for _, sub := range existing {
		if strings.EqualFold(sub.Name, name) && fmt.Sprintf("%.2f", sub.Cost) == price {
			return true
		}
	}
	return false
}

func (h *ImportHandler) getOrCreateCategory(name string) *models.Category {
	categories, err := h.categoryService.GetAll()
	if err != nil {
		return nil
	}

	for _, cat := range categories {
		if strings.EqualFold(cat.Name, name) {
			return &cat
		}
	}

	newCat := &models.Category{Name: name}
	created, err := h.categoryService.Create(newCat)
	if err != nil {
		slog.Error("failed to create category", "category", name, "error", err)
		return nil
	}
	return created
}

// ImportEncrypted handles importing from an AES-256-GCM encrypted backup file (.stbk)
func (h *ImportHandler) ImportEncrypted(c *gin.Context) {
	password := c.PostForm("password")
	if password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrPasswordRequired})
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrNoFileUploaded})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrFailedReadFile})
		return
	}

	decrypted, err := crypto.Decrypt(data, password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Decryption failed: wrong password or corrupted file"})
		return
	}

	// Re-import using the SubTrackr format
	result := h.importSubTrackr(decrypted)

	c.HTML(http.StatusOK, "import-result.html", gin.H{
		"Result": result,
	})
}
