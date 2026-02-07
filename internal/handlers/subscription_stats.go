package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetStats returns current statistics
func (h *SubscriptionHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
