package logging

import (
	"sync"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
)

const Capacity = 100

type Log struct {
	ringPointer  int
	reportBuffer [Capacity]Report //ring buffer
	mtx          sync.RWMutex
}

type Report struct {
	report     *function.ExecutionReport
	expiration int64
}
type LogStatus struct {
	AvgWarmInitTime      float64
	AvgColdInitTime      float64
	AvgExecutionTime     float64
	AvgOffloadingLatency float64
}

func (l *Log) Update(e *function.ExecutionReport) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	//insert new Report in the ring
	l.reportBuffer[l.ringPointer].report = e
	l.reportBuffer[l.ringPointer].expiration = time.Now().Add(time.Duration(config.GetInt("logging.expiration", 3)) * time.Minute).UnixNano()
	l.ringPointer = (l.ringPointer + 1) % Capacity
}

func (l *Log) GetLogStatus() (status *LogStatus) {
	l.mtx.RLock()
	defer l.mtx.RUnlock()
	var warmInit, coldInit, executionTime, latency float64
	var coldInitCounter, warmInitCounter int

	for _, reportItem := range l.reportBuffer {
		if reportItem.report == nil {
			continue
		}

		if reportItem.report.IsWarmStart {
			warmInit += reportItem.report.InitTime
			warmInitCounter++
		} else {
			coldInit += reportItem.report.InitTime
			coldInitCounter++
		}
		executionTime += reportItem.report.Duration
		latency += reportItem.report.OffloadLatency
	}

	return &LogStatus{
		warmInit / float64(warmInitCounter),
		coldInit / float64(coldInitCounter),
		executionTime / float64(coldInitCounter+warmInitCounter),
		latency / float64(coldInitCounter+warmInitCounter),
	}

}

func (l *Log) CleanupExpiredReports() {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	now := time.Now().UnixNano()
	for _, reportItem := range l.reportBuffer {
		if reportItem.report != nil && now > reportItem.expiration {
			reportItem.report = nil
		}
	}
}
