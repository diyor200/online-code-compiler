package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) RegisterRoutes() {
	h.Engine.GET("/health", h.healthCheck)
	v1 := h.Engine.Group("/api/v1")

	v1.GET("/languages", h.getSupportedLanguages)
	v1.POST("/execute", h.executeCode)
}

func (h *Handler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
