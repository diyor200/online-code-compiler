package scheme

import "github.com/diyor200/code-compiler/internal/domain"

type CodeRequest struct {
	Code     string `json:"code" binding:"required"`
	Language string `json:"lang" binding:"required"`
}

func (c *CodeRequest) ToModel() domain.ExecCode {
	return domain.ExecCode{
		Lang: c.Language,
		Code: c.Code,
	}
}

type CodeResponse struct {
	Result string `json:"result"`
	Error  error  `json:"error"`
}

func ToExecCodeResponse(result domain.ExecResult) *CodeResponse {
	return &CodeResponse{
		Result: result.Result,
		Error:  result.Error,
	}
}
