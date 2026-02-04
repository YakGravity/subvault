package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
)

func setupCategoryTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&models.Category{}, &models.Subscription{})
	assert.NoError(t, err)

	return db
}

func TestCategoryService_DeleteDefault(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := repository.NewCategoryRepository(db)
	svc := NewCategoryService(repo)

	// Create default category
	defaultCat := models.Category{Name: "General", IsDefault: true}
	db.Create(&defaultCat)

	// Try to delete - should fail
	err := svc.Delete(defaultCat.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete default category")
}

func TestCategoryService_DeleteNonDefault(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := repository.NewCategoryRepository(db)
	svc := NewCategoryService(repo)

	// Create default category
	defaultCat := models.Category{Name: "General", IsDefault: true}
	db.Create(&defaultCat)

	// Create non-default category
	otherCat := models.Category{Name: "Streaming", IsDefault: false}
	db.Create(&otherCat)

	// Create subscription linked to the non-default category
	sub := models.Subscription{
		Name:       "Netflix",
		Cost:       12.99,
		Schedule:   "Monthly",
		Status:     "Active",
		CategoryID: otherCat.ID,
	}
	db.Create(&sub)

	// Delete non-default category - should succeed
	err := svc.Delete(otherCat.ID)
	assert.NoError(t, err)

	// Verify subscription was reassigned to default category
	var updatedSub models.Subscription
	db.First(&updatedSub, sub.ID)
	assert.Equal(t, defaultCat.ID, updatedSub.CategoryID)

	// Verify the non-default category was deleted
	var count int64
	db.Model(&models.Category{}).Where("id = ?", otherCat.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestCategoryService_GetDefault(t *testing.T) {
	db := setupCategoryTestDB(t)
	repo := repository.NewCategoryRepository(db)
	svc := NewCategoryService(repo)

	// Create default category
	defaultCat := models.Category{Name: "General", IsDefault: true}
	db.Create(&defaultCat)

	// Create non-default category
	otherCat := models.Category{Name: "Streaming", IsDefault: false}
	db.Create(&otherCat)

	// Get default - should return the default category
	result, err := svc.GetDefault()
	assert.NoError(t, err)
	assert.Equal(t, defaultCat.ID, result.ID)
	assert.Equal(t, "General", result.Name)
	assert.True(t, result.IsDefault)
}
