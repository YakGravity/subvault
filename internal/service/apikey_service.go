package service

import (
	"subvault/internal/models"
	"subvault/internal/repository"
)

type APIKeyService struct {
	repo *repository.SettingsRepository
}

func NewAPIKeyService(repo *repository.SettingsRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

// CreateAPIKey creates a new API key
func (a *APIKeyService) CreateAPIKey(name, key string) (*models.APIKey, error) {
	apiKey := &models.APIKey{
		Name: name,
		Key:  key,
	}
	return a.repo.CreateAPIKey(apiKey)
}

// GetAllAPIKeys retrieves all API keys
func (a *APIKeyService) GetAllAPIKeys() ([]models.APIKey, error) {
	return a.repo.GetAllAPIKeys()
}

// DeleteAPIKey deletes an API key
func (a *APIKeyService) DeleteAPIKey(id uint) error {
	return a.repo.DeleteAPIKey(id)
}

// ValidateAPIKey checks if an API key is valid and updates usage
func (a *APIKeyService) ValidateAPIKey(key string) (*models.APIKey, error) {
	apiKey, err := a.repo.GetAPIKeyByKey(key)
	if err != nil {
		return nil, err
	}

	// Update usage stats
	err = a.repo.UpdateAPIKeyUsage(apiKey.ID)
	if err != nil {
		return nil, err
	}

	return apiKey, nil
}
