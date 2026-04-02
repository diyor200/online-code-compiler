package domain

type ExecCode struct {
	Lang string
	Code string
}

type ExecResult struct {
	Result string
	Error  error
}
