package logging

import (
	"log"
	"sync"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
)

// Logger keeps track of statistical parameters
type Logger struct {
	logs map[string]*functionLog
	mtx  *sync.RWMutex
}

type functionLog struct {
	remoteLog *Log                           // take info regarded to the remote server
	localLog  *Log                           // take local information
	ch        chan *function.ExecutionReport // to handle incoming reports
	stop      chan bool                      // to stop logging
}

var (
	logger *Logger
)

var lock = &sync.Mutex{}

// GetLogger singleton implementation of a Logger
func GetLogger() (l *Logger) {
	lock.Lock()
	defer lock.Unlock()

	if logger == nil {

		logger = &Logger{make(map[string]*functionLog), &sync.RWMutex{}} // <-- thread safe
	}

	return logger
}

func (logger *Logger) Exists(functionName string) bool {
	logger.mtx.RLock()
	defer logger.mtx.RUnlock()
	if _, ok := logger.logs[functionName]; !ok {
		return false
	}
	return true
}

func (logger *Logger) InsertNewLog(functionName string) {
	logger.mtx.Lock()
	defer logger.mtx.Unlock()
	logger.logs[functionName] = &functionLog{
		new(Log),
		new(Log),
		make(chan *function.ExecutionReport, 100), // buffered channel, todo change capacity
		make(chan bool),
	}
	// run logging actions async
	go logger.logs[functionName].run()
}

// CleanUpLog deletes all information stored into the logging
func (logger *Logger) CleanUpLog() {
	logger.mtx.Lock()
	defer logger.mtx.Unlock()

	for k := range logger.logs {
		//stop the corresponding thread
		logger.logs[k].stop <- true
		//remove the logging from the map
		delete(logger.logs, k)
	}
}

//GetLocalLogStatus returns a struct containing all statistical parameters
func (logger *Logger) GetLocalLogStatus(functionName string) (status *LogStatus, e error) {
	logger.mtx.RLock()
	defer logger.mtx.RUnlock()
	if _, ok := logger.logs[functionName]; !ok {
		return nil, NotExistingLog
	}

	return logger.logs[functionName].localLog.GetLogStatus(), nil
}

//GetRemoteLogStatus returns a struct containing all statistical parameters
func (logger *Logger) GetRemoteLogStatus(functionName string) (status *LogStatus, e error) {
	logger.mtx.RLock()
	defer logger.mtx.RUnlock()
	if _, ok := logger.logs[functionName]; !ok {
		return nil, NotExistingLog
	}

	return logger.logs[functionName].remoteLog.GetLogStatus(), nil
}

//SendReport send the Report to the correct logging, asynchronous update
func (logger *Logger) SendReport(report *function.ExecutionReport, functionName string) error {
	logger.mtx.RLock()
	defer logger.mtx.RUnlock()

	if _, ok := logger.logs[functionName]; ok {
		select {
		case logger.logs[functionName].ch <- report:
			return nil
		default:
			return GeneralError
		}
	}

	return NotExistingLog
}

func (logPtr *functionLog) run() {
	ticker := time.NewTicker(time.Duration(config.GetInt("logging.cleanInterval", 3)) * time.Minute)
	for {
		select {
		case <-logPtr.stop:
			ticker.Stop()
			return
		case report := <-logPtr.ch:
			logPtr.updateReport(report)
		case <-ticker.C:
			logPtr.remoteLog.CleanupExpiredReports()
			logPtr.localLog.CleanupExpiredReports()
		}
	}
}

// updateReport logging update using parameters included within the Report
func (logPtr *functionLog) updateReport(report *function.ExecutionReport) {
	logger.mtx.RLock()
	defer logger.mtx.RUnlock()

	if report.OffloadLatency != 0 {
		// if latency is nonzero the executions takes remotely
		logPtr.remoteLog.Update(report)
		log.Printf("remote logging: %v\n", report)
	} else {
		// if latency is zero the executions takes locally
		logPtr.localLog.Update(report)
		log.Printf(" local logging: %v\n", report)
	}

}
