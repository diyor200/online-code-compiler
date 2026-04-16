package domain

// ExecutorConfig holds configuration for code execution
type ExecutorConfig struct {
	TimeoutSeconds  int64
	MemoryLimit     int64 // in bytes
	CPUQuota        int64 // CPU quota in microseconds
	NetworkDisabled bool
}

// ExecutionResult holds result of execution
type ExecutionResult struct {
	Stdout        string
	Stderr        string
	ExitCode      int
	ExecutionTime float64
	Error         error
}

type StreamWriter interface {
	Stdout(line string)
	Stderr(line string)
	Error(err error)
	Done(code int)
}
