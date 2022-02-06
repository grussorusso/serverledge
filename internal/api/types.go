package api

type FunctionInvocationRequest struct {
	Params      map[string]string
	QoSClass    string
	QoSMaxRespT float64
}
