package bootstrap

import (
	"github.com/diyor200/code-compiler/internal/domain"
	"github.com/diyor200/code-compiler/internal/usecase/executor"
	"github.com/docker/docker/client"
)

type UseCaseBuilder struct {
	executor *executor.UseCase
}

func NewUseCaseBuilder(cl *client.Client) *UseCaseBuilder {
	cfg := domain.ExecutorConfig{
		TimeoutSeconds:  30,
		MemoryLimit:     512 * 1024 * 1024, // 512 MB
		CPUQuota:        100000,            // 1 CPU core
		NetworkDisabled: true,
	}
	return &UseCaseBuilder{
		executor: executor.New(cl, cfg),
	}
}
