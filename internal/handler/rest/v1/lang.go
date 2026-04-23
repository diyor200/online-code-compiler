package v1

import (
	"github.com/diyor200/code-compiler/internal/domain"
	"github.com/diyor200/code-compiler/internal/handler/rest/v1/scheme"
	"github.com/gin-gonic/gin"
	"net/http"
)

// getSupportedLanguages godoc
// @Summary Get supported languages
// @Description Get supported languages
// @Produce json
// @Success 200 {object} scheme.LanguageSettings
// @Router /api/v1/languages [GET]
func (h *Handler) getSupportedLanguages(c *gin.Context) {
	c.JSON(http.StatusOK, scheme.LanguageSettings{Supported: domain.SupportedLangs})
}
