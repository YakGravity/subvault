package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ListAPIKeys returns all API keys
func (h *SettingsHandler) ListAPIKeys(c *gin.Context) {
	keys, err := h.apiKey.GetAllAPIKeys()
	if err != nil {
		slog.Error("failed to list API keys", "error", err)
		c.HTML(http.StatusInternalServerError, "api-keys-list.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": "An internal error occurred",
		}))
		return
	}

	// Don't send the actual key values for existing keys
	for i := range keys {
		if !keys[i].IsNew {
			keys[i].Key = ""
		}
	}

	c.HTML(http.StatusOK, "api-keys-list.html", mergeTemplateData(baseTemplateData(c), gin.H{
		"Keys": keys,
	}))
}

// CreateAPIKey generates a new API key
func (h *SettingsHandler) CreateAPIKey(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		c.HTML(http.StatusBadRequest, "api-keys-list.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": "API key name is required",
		}))
		return
	}

	// Generate a secure random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		c.HTML(http.StatusInternalServerError, "api-keys-list.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": "Failed to generate API key",
		}))
		return
	}

	apiKey := "sk_" + hex.EncodeToString(keyBytes)

	// Save the API key
	newKey, err := h.apiKey.CreateAPIKey(name, apiKey)
	if err != nil {
		slog.Error("failed to create API key", "error", err)
		c.HTML(http.StatusInternalServerError, "api-keys-list.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": "An internal error occurred",
		}))
		return
	}

	// Get all keys including the new one
	keys, err := h.apiKey.GetAllAPIKeys()
	if err != nil {
		slog.Error("failed to list API keys after creation", "error", err)
		c.HTML(http.StatusInternalServerError, "api-keys-list.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": "An internal error occurred",
		}))
		return
	}

	// Mark the new key and include its value
	for i := range keys {
		if keys[i].ID == newKey.ID {
			keys[i].IsNew = true
			keys[i].Key = apiKey
		} else {
			keys[i].Key = ""
		}
	}

	c.HTML(http.StatusOK, "api-keys-list.html", mergeTemplateData(baseTemplateData(c), gin.H{
		"Keys": keys,
	}))
}

// DeleteAPIKey removes an API key
func (h *SettingsHandler) DeleteAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "api-keys-list.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": "Invalid API key ID",
		}))
		return
	}

	err = h.apiKey.DeleteAPIKey(uint(id))
	if err != nil {
		slog.Error("failed to delete API key", "error", err, "id", id)
		c.HTML(http.StatusInternalServerError, "api-keys-list.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": "An internal error occurred",
		}))
		return
	}

	// Return updated list
	h.ListAPIKeys(c)
}
