package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPageLimit = 50
	maxPageLimit     = 100
)

// Common error messages used across handlers
const (
	ErrInvalidID            = "Invalid ID"
	ErrSubscriptionNotFound = "Subscription not found"
	ErrCategoryNotFound     = "Category not found"
	ErrPasswordRequired     = "Password required"
	ErrNoFileUploaded       = "No file uploaded"
	ErrFailedReadFile       = "Failed to read file"
	ErrPasswordsDoNotMatch  = "Passwords do not match"
	ErrInvalidRequestBody   = "Invalid request body"
	ErrInternalServer       = "Internal server error"
)

// APIErrorResponse is the standard error format for all API v1 endpoints.
type APIErrorResponse struct {
	Error string `json:"error"`
}

// apiError sends a standardized JSON error response for API endpoints.
func apiError(c *gin.Context, status int, message string) {
	c.JSON(status, APIErrorResponse{Error: message})
}

// apiBadRequest sends a 400 Bad Request error.
func apiBadRequest(c *gin.Context, message string) {
	apiError(c, http.StatusBadRequest, message)
}

// apiNotFound sends a 404 Not Found error.
func apiNotFound(c *gin.Context, message string) {
	apiError(c, http.StatusNotFound, message)
}

// apiInternalError sends a 500 Internal Server Error.
func apiInternalError(c *gin.Context, message string) {
	apiError(c, http.StatusInternalServerError, message)
}

// PaginationMeta contains pagination metadata for list responses.
type PaginationMeta struct {
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Total  int64 `json:"total"`
}

// PaginatedResponse wraps list data with pagination metadata.
type PaginatedResponse struct {
	Data       any            `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// parsePagination extracts and validates limit/offset from query params.
func parsePagination(c *gin.Context) (limit, offset int) {
	limit = defaultPageLimit
	offset = 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}
