package domain

type ExecuteRequest struct {
	Language string
	Code     string
	Stdin    string
}

// ExecuteResponse represents the API response
type ExecuteResponse struct {
	Stdout        string
	Stderr        string
	ExitCode      int
	ExecutionTime float64
}
