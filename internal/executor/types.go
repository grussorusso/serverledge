package executor

type InvocationRequest struct {
	Command    []string
	Params     map[string]string
	Handler    string
	HandlerDir string
}

type InvocationResult struct {
	Success  bool
	Result   string
	Duration float64
	CPUTime  float64
}
