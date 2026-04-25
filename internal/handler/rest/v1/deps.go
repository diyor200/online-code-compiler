package v1

import (
	"context"

	"github.com/diyor200/code-compiler/internal/domain"

	"github.com/diyor200/code-compiler/internal/config"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Engine   *gin.Engine
	conf     *config.Config
	executor Executor
}

func NewHandler(engine *gin.Engine, conf *config.Config, executor Executor) *Handler {
	h := &Handler{
		Engine:   engine,
		conf:     conf,
		executor: executor,
	}

	h.RegisterRoutes()

	return h
}

type Executor interface {
	Execute(ctx context.Context, taskID string, writer domain.StreamWriter)
	CreateTask(ctx context.Context, data domain.ExecuteRequest) (string, error)
}
