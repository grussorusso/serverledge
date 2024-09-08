package fc_scheduling

import (
	"github.com/grussorusso/serverledge/internal/fc"
	"time"
)

// TODO: offload the entire node when is cloud only
func SubmitCompositionRequest(fcReq *fc.CompositionRequest) error {
	executionReport, err := fcReq.Fc.Invoke(fcReq)
	if err != nil {
		return err
	}
	fcReq.ExecReport = executionReport
	fcReq.ExecReport.ResponseTime = time.Now().Sub(fcReq.Arrival).Seconds()
	return nil
}

// TODO: offload the entire node.
// TODO: make sure the requestId is the one returned from the serverledge node that will execute
func SubmitAsyncCompositionRequest(fcReq *fc.CompositionRequest) {
	executionReport, errInvoke := fcReq.Fc.Invoke(fcReq)
	if errInvoke != nil {
		PublishAsyncCompositionResponse(fcReq.ReqId, fc.CompositionResponse{Success: false})
		return
	}
	PublishAsyncCompositionResponse(fcReq.ReqId, fc.CompositionResponse{
		Success:      true,
		Result:       fcReq.ExecReport.Result,
		Reports:      fcReq.ExecReport.String(),
		ResponseTime: fcReq.ExecReport.ResponseTime,
	})
	fcReq.ExecReport = executionReport
	fcReq.ExecReport.ResponseTime = time.Now().Sub(fcReq.Arrival).Seconds()
}
