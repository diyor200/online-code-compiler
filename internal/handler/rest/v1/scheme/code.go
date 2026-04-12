package scheme

import "github.com/diyor200/code-compiler/internal/domain"

type ExecuteRequest struct {
	Language string `form:"language" binding:"required"`
	Code     string `form:"code" binding:"required"`
	Stdin    string `form:"stdin"`
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
