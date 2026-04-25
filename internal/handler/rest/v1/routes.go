package v1

import (
	"net/http"

	_ "github.com/diyor200/code-compiler/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes() {
	h.Engine.GET("/health", h.healthCheck)
	h.Engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	v1 := h.Engine.Group("/api/v1")

	v1.GET("/languages", h.getSupportedLanguages)
	v1.POST("/task", h.createTask)
	v1.GET("/task/:id", h.getTaskResult)
}

func (h *Handler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
