package handlers

import (
	"subtrackr/internal/service"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	service service.SettingsServiceInterface
}

func NewSettingsHandler(service service.SettingsServiceInterface) *SettingsHandler {
	return &SettingsHandler{service: service}
}

// settingsBaseData returns common template data for all settings pages
func (h *SettingsHandler) settingsBaseData(c *gin.Context, currentTab string) gin.H {
	data := baseTemplateData(c)
	mergeTemplateData(data, gin.H{
		"CurrentPage": "settings",
		"CurrentTab":  currentTab,
	})
	return data
}
