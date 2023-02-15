package executor

import "github.com/grussorusso/serverledge/internal/function"

type InvocationRequest struct {
	OriginalRequest function.Request
	Id              string
	Command         []string
	Params          map[string]interface{}
	Handler         string
	HandlerDir      string
	NodeIP          string
	Async           bool
	Class           int64
}

type InvocationResult struct {
	Success bool
	Result  string
	QoS     int64
	Id      string
}

type FallbackAcquisitionRequest struct {
	FallbackAddresses []string
}

type FallbackAcquisitionResult struct {
	Success bool
}

type MigrationResult struct {
	Result  string
	Success bool
	Class   int64
	Id      string
	Error   error
}
