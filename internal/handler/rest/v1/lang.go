package v1

import (
	"github.com/diyor200/code-compiler/internal/domain"
	"github.com/diyor200/code-compiler/internal/handler/rest/v1/scheme"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) getSupportedLanguages(c *gin.Context) {
	c.JSON(http.StatusOK, scheme.LanguageSettings{Supported: domain.SupportedLangs})
}
