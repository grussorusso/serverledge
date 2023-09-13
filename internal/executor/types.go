package executor

// InvocationRequest is a struct used by the executor to effectively run the function
type InvocationRequest struct {
	Command         []string
	Params          map[string]interface{}
	Handler         string
	HandlerDir      string
	IsInComposition bool
}

type InvocationResult struct {
	Success bool
	Result  string
}
