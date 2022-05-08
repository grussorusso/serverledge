package executor

type InvocationRequest struct {
	Command    []string
	Params     map[string]interface{}
	Handler    string
	HandlerDir string
}

type InvocationResult struct {
	Success bool
	Result  string
}
