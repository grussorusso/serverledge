package fc_scheduling

import (
	"github.com/grussorusso/serverledge/internal/fc"
	"time"
)

func SubmitCompositionRequest(fcReq *fc.CompositionRequest) error {
	executionReport, err := fcReq.Fc.Invoke(fcReq)
	if err != nil {
		return err
	}
	fcReq.ExecReport = executionReport
	fcReq.ExecReport.ResponseTime = time.Now().Sub(fcReq.Arrival).Seconds()
	return nil
}

func SubmitAsyncCompositionRequest(fcReq *fc.CompositionRequest) {
	executionReport, errInvoke := fcReq.Fc.Invoke(fcReq)
	if errInvoke != nil {
		PublishAsyncCompositionResponse(fcReq.ReqId, fc.CompositionResponse{Success: false})
	}
	PublishAsyncCompositionResponse(fcReq.ReqId, fc.CompositionResponse{Success: true, CompositionExecutionReport: executionReport})
	fcReq.ExecReport = executionReport
	fcReq.ExecReport.ResponseTime = time.Now().Sub(fcReq.Arrival).Seconds()
}
