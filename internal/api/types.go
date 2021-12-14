package api

type FunctionCreationRequest struct {
	Name            string
	Runtime         string
	Memory          int
	SourceTarBase64 string
	Handler         string
}

type FunctionInvocationRequest struct {
	Params      map[string]string
	QoSClass    string
	QoSMaxRespT float64
}
