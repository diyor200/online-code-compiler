package bootstrap

import "github.com/diyor200/code-compiler/internal/usecase/executor"

type UseCaseBuilder struct {
	executor *executor.UseCase
}

func NewUseCaseBuilder() *UseCaseBuilder {
	return &UseCaseBuilder{
		executor: executor.New(),
	}
}
