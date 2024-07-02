package executor

type InvocationRequest struct {
	Command      []string
	Params       map[string]interface{}
	Handler      string
	HandlerDir   string
	ReturnOutput bool
}

type InvocationResult struct {
	Success bool
	Result  string
	Output  string
}
