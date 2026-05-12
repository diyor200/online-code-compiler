package v1

import (
	"net/http"

	_ "github.com/diyor200/code-compiler/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (h *Handler) RegisterRoutes() {
	h.Engine.GET("/health", h.healthCheck)
	h.Engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	h.Engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	v1 := h.Engine.Group("/api/v1")

	// rate limited routes
	limited := v1.Group("/")
	limited.Use(h.rateLimitMiddleware())
	{
		limited.GET("/languages", h.getSupportedLanguages)
		limited.POST("/task", h.createTask)
	}

	v1.GET("/task/:id", h.getTaskResult)
}

func (h *Handler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
