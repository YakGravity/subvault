package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetStats returns current statistics
func (h *SubscriptionHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		slog.Error("failed to get stats", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
