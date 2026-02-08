package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"subvault/internal/models"
	"subvault/internal/service"

	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	service service.CategoryServiceInterface
}

func NewCategoryHandler(service service.CategoryServiceInterface) *CategoryHandler {
	return &CategoryHandler{service: service}
}

// List all categories
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	categories, err := h.service.GetAll()
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, categories)
}

// Create a new category
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var category models.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := h.service.Create(&category)
	if err != nil {
		slog.Error("failed to create category", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusCreated, created)
}

// Update a category
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidID})
		return
	}
	var category models.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.service.Update(uint(id), &category)
	if err != nil {
		slog.Error("failed to update category", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, updated)
}

// Delete a category
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidID})
		return
	}
	if err := h.service.Delete(uint(id)); err != nil {
		if strings.Contains(err.Error(), "cannot delete default category") {
			c.JSON(http.StatusBadRequest, gin.H{"error": tr(c, "category_cannot_delete_default", "Cannot delete the default category")})
			return
		}
		slog.Error("failed to delete category", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.Status(http.StatusNoContent)
}
