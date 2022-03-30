package client

type InvocationRequest struct {
	Params          map[string]string
	QoSClass        int64
	QoSMaxRespT     float64
	CanDoOffloading bool
}
