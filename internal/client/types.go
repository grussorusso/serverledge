package client

type InvocationRequest struct {
	Params          map[string]interface{}
	QoSClass        int64
	QoSMaxRespT     float64
	CanDoOffloading bool
	Async           bool
}
