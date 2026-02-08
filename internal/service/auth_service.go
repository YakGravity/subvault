package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log/slog"
	"subvault/internal/repository"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	settings *SettingsService
	repo     *repository.SettingsRepository
}

func NewAuthService(settings *SettingsService, repo *repository.SettingsRepository) *AuthService {
	return &AuthService{
		settings: settings,
		repo:     repo,
	}
}

// IsAuthEnabled returns whether authentication is enabled
func (a *AuthService) IsAuthEnabled() bool {
	return a.settings.GetBoolSettingWithDefault(SettingKeyAuthEnabled, false)
}

// SetAuthEnabled enables or disables authentication
func (a *AuthService) SetAuthEnabled(enabled bool) error {
	return a.settings.SetBoolSetting(SettingKeyAuthEnabled, enabled)
}

// GetAuthUsername returns the configured admin username
func (a *AuthService) GetAuthUsername() (string, error) {
	val, ok := a.settings.GetCached(SettingKeyAuthUsername)
	if !ok {
		return "", fmt.Errorf("auth_username not found")
	}
	return val, nil
}

// SetAuthUsername sets the admin username
func (a *AuthService) SetAuthUsername(username string) error {
	defer a.settings.InvalidateCache()
	return a.repo.Set(SettingKeyAuthUsername, username)
}

// HashPassword hashes a password using bcrypt
func (a *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// SetAuthPassword hashes and stores the admin password
func (a *AuthService) SetAuthPassword(password string) error {
	hash, err := a.HashPassword(password)
	if err != nil {
		return err
	}
	defer a.settings.InvalidateCache()
	return a.repo.Set(SettingKeyAuthPasswordHash, hash)
}

// ValidatePassword checks if a password matches the stored hash
func (a *AuthService) ValidatePassword(password string) error {
	hash, ok := a.settings.GetCached(SettingKeyAuthPasswordHash)
	if !ok {
		return fmt.Errorf("no password configured")
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GetOrGenerateSessionSecret returns the session secret, generating one if it doesn't exist
func (a *AuthService) GetOrGenerateSessionSecret() (string, error) {
	secret, ok := a.settings.GetCached(SettingKeyAuthSessionSecret)
	if ok && secret != "" {
		return secret, nil
	}

	// Generate a new 64-byte random secret
	bytes := make([]byte, 64)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	secret = base64.URLEncoding.EncodeToString(bytes)

	// Save it
	if err := a.repo.Set(SettingKeyAuthSessionSecret, secret); err != nil {
		return "", err
	}
	a.settings.InvalidateCache()

	return secret, nil
}

// GetOrGenerateCSRFSecret returns the CSRF secret, generating one if it doesn't exist
func (a *AuthService) GetOrGenerateCSRFSecret() ([]byte, error) {
	secret, ok := a.settings.GetCached(SettingKeyCSRFSecret)
	if ok && secret != "" {
		decoded, err := base64.URLEncoding.DecodeString(secret)
		if err != nil {
			return nil, err
		}
		return decoded, nil
	}

	// Generate a new 32-byte random secret
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	encoded := base64.URLEncoding.EncodeToString(bytes)

	if err := a.repo.Set(SettingKeyCSRFSecret, encoded); err != nil {
		return nil, err
	}
	a.settings.InvalidateCache()

	return bytes, nil
}

// SetupAuth sets up authentication with username and password
func (a *AuthService) SetupAuth(username, password string) error {
	// Set username
	if err := a.SetAuthUsername(username); err != nil {
		return err
	}

	// Set password
	if err := a.SetAuthPassword(password); err != nil {
		return err
	}

	// Generate session secret
	if _, err := a.GetOrGenerateSessionSecret(); err != nil {
		return err
	}

	// Enable auth
	return a.SetAuthEnabled(true)
}

// DisableAuth disables authentication and removes credentials
func (a *AuthService) DisableAuth() error {
	// Disable auth first
	if err := a.SetAuthEnabled(false); err != nil {
		return err
	}

	return nil
}

// GenerateResetToken generates a password reset token
func (a *AuthService) GenerateResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(bytes)

	if err := a.repo.Set(SettingKeyAuthResetToken, token); err != nil {
		return "", err
	}

	expiry := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	if err := a.repo.Set(SettingKeyAuthResetExpiry, expiry); err != nil {
		return "", err
	}

	a.settings.InvalidateCache()
	return token, nil
}

// ValidateResetToken checks if a reset token is valid
func (a *AuthService) ValidateResetToken(token string) error {
	storedToken, ok := a.settings.GetCached(SettingKeyAuthResetToken)
	if !ok || subtle.ConstantTimeCompare([]byte(storedToken), []byte(token)) != 1 {
		return fmt.Errorf("invalid token")
	}

	expiryStr, ok := a.settings.GetCached(SettingKeyAuthResetExpiry)
	if !ok {
		return fmt.Errorf("token expired")
	}

	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil || time.Now().After(expiry) {
		return fmt.Errorf("token expired")
	}

	return nil
}

// ClearResetToken removes the reset token after use
func (a *AuthService) ClearResetToken() error {
	a.repo.Delete(SettingKeyAuthResetToken)
	a.repo.Delete(SettingKeyAuthResetExpiry)
	a.settings.InvalidateCache()
	slog.Debug("reset token cleared")
	return nil
}
