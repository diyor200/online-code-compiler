package scheme

import "github.com/diyor200/code-compiler/internal/domain"

type ExecuteRequest struct {
	Language string `json:"language" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Stdin    string `json:"stdin"`
}

// ExecuteResponse represents the API response
type ExecuteResponse struct {
	Stdout        string  `json:"stdout"`
	Stderr        string  `json:"stderr"`
	ExitCode      int     `json:"exitCode"`
	ExecutionTime float64 `json:"executionTime"`
}

func (c *ExecuteRequest) ToModel() domain.ExecuteRequest {
	return domain.ExecuteRequest{
		Language: c.Language,
		Code:     c.Code,
		Stdin:    c.Stdin,
	}
}

func ToExecCodeResponse(result domain.ExecuteResponse) *ExecuteResponse {
	return &ExecuteResponse{
		Stdout:        result.Stdout,
		Stderr:        result.Stderr,
		ExitCode:      result.ExitCode,
		ExecutionTime: result.ExecutionTime,
	}
}
