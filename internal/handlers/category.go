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

// ListCategories returns all categories with pagination support.
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	limit, offset := parsePagination(c)

	categories, total, err := h.service.GetAllPaginated(limit, offset)
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		apiInternalError(c, "Failed to retrieve categories")
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Data: categories,
		Pagination: PaginationMeta{
			Limit:  limit,
			Offset: offset,
			Total:  total,
		},
	})
}

// Create a new category
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var category models.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		apiBadRequest(c, ErrInvalidRequestBody)
		return
	}
	created, err := h.service.Create(&category)
	if err != nil {
		slog.Error("failed to create category", "error", err)
		apiInternalError(c, "Failed to create category")
		return
	}
	c.JSON(http.StatusCreated, created)
}

// Update a category
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apiBadRequest(c, ErrInvalidID)
		return
	}
	var category models.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		apiBadRequest(c, ErrInvalidRequestBody)
		return
	}
	updated, err := h.service.Update(uint(id), &category)
	if err != nil {
		slog.Error("failed to update category", "error", err, "id", id)
		apiInternalError(c, "Failed to update category")
		return
	}
	c.JSON(http.StatusOK, updated)
}

// Delete a category
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apiBadRequest(c, ErrInvalidID)
		return
	}
	if err := h.service.Delete(uint(id)); err != nil {
		if strings.Contains(err.Error(), "cannot delete default category") {
			apiBadRequest(c, tr(c, "category_cannot_delete_default", "Cannot delete the default category"))
			return
		}
		slog.Error("failed to delete category", "error", err, "id", id)
		apiInternalError(c, "Failed to delete category")
		return
	}
	c.Status(http.StatusNoContent)
}
