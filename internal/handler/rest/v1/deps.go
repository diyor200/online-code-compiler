package v1

import (
	"context"
	"github.com/diyor200/code-compiler/pkg/rate_limiter"

	"github.com/diyor200/code-compiler/internal/domain"

	"github.com/diyor200/code-compiler/internal/config"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Engine      *gin.Engine
	conf        *config.Config
	executor    Executor
	rateLimiter rate_limiter.RateLimiter
}

func NewHandler(
	engine *gin.Engine,
	conf *config.Config,
	executor Executor,
	rateLimiter rate_limiter.RateLimiter,
) *Handler {
	h := &Handler{
		Engine:      engine,
		conf:        conf,
		executor:    executor,
		rateLimiter: rateLimiter,
	}

	h.Engine.Use(h.corsMiddleware())
	h.RegisterRoutes()

	return h
}

type Executor interface {
	Execute(ctx context.Context, taskID string, writer domain.StreamWriter)
	CreateTask(ctx context.Context, data domain.ExecuteRequest) (string, error)
}
