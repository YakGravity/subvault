package repository

import (
	"subvault/internal/models"

	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(category *models.Category) (*models.Category, error) {
	if err := r.db.Create(category).Error; err != nil {
		return nil, err
	}
	return category, nil
}

func (r *CategoryRepository) GetAll() ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.Order("name ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// GetAllPaginated returns categories with pagination support.
func (r *CategoryRepository) GetAllPaginated(limit, offset int) ([]models.Category, int64, error) {
	var total int64
	if err := r.db.Model(&models.Category{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var categories []models.Category
	if err := r.db.Order("name ASC").Limit(limit).Offset(offset).Find(&categories).Error; err != nil {
		return nil, 0, err
	}
	return categories, total, nil
}

func (r *CategoryRepository) GetByID(id uint) (*models.Category, error) {
	var category models.Category
	if err := r.db.First(&category, id).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) Update(id uint, category *models.Category) (*models.Category, error) {
	if err := r.db.Model(&models.Category{}).Where("id = ?", id).Updates(category).Error; err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

func (r *CategoryRepository) Delete(id uint) error {
	return r.db.Delete(&models.Category{}, id).Error
}

func (r *CategoryRepository) HasSubscriptions(id uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Subscription{}).Where("category_id = ?", id).Count(&count).Error
	return count > 0, err
}

func (r *CategoryRepository) GetDefault() (*models.Category, error) {
	var category models.Category
	result := r.db.Where("is_default = ?", true).First(&category)
	if result.Error != nil {
		return nil, result.Error
	}
	return &category, nil
}

func (r *CategoryRepository) ReassignSubscriptions(fromID, toID uint) error {
	return r.db.Model(&models.Subscription{}).Where("category_id = ?", fromID).Update("category_id", toID).Error
}
