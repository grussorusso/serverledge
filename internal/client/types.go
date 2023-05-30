package client

type InvocationRequest struct {
	Params          map[string]interface{}
	QoSClass        string
	QoSMaxRespT     float64
	CanDoOffloading bool
	Async           bool
}
