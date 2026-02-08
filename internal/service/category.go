package service

import (
	"fmt"
	"subvault/internal/models"
	"subvault/internal/repository"
)

// CategoryService provides business logic for categories
type CategoryService struct {
	repo *repository.CategoryRepository
}

func NewCategoryService(repo *repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) Create(category *models.Category) (*models.Category, error) {
	return s.repo.Create(category)
}

func (s *CategoryService) GetAll() ([]models.Category, error) {
	return s.repo.GetAll()
}

func (s *CategoryService) GetByID(id uint) (*models.Category, error) {
	return s.repo.GetByID(id)
}

func (s *CategoryService) Update(id uint, category *models.Category) (*models.Category, error) {
	return s.repo.Update(id, category)
}

func (s *CategoryService) Delete(id uint) error {
	category, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if category.IsDefault {
		return fmt.Errorf("cannot delete default category")
	}
	defaultCat, err := s.repo.GetDefault()
	if err != nil {
		return fmt.Errorf("failed to find default category: %w", err)
	}
	if err := s.repo.ReassignSubscriptions(id, defaultCat.ID); err != nil {
		return fmt.Errorf("failed to reassign subscriptions: %w", err)
	}
	return s.repo.Delete(id)
}

func (s *CategoryService) GetDefault() (*models.Category, error) {
	return s.repo.GetDefault()
}
