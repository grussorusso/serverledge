package scheduling

import (
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
	"log"
	"sync"
	"time"
)

const (
	DROP_REQUEST           = 0
	EXEC_LOCAL_REQUEST     = 1
	EXEC_CLOUD_REQUEST     = 2
	EXEC_NEIGHBOUR_REQUEST = 3
)

const rttPoints = 10
const initPoints = 10

type functionInfo struct {
	name string
	//Number of function requests
	count [2]int
	//Mean response time
	meanResponse [2]float64
	//Variance of the response time
	varianceResponse [2]float64
	//Number of requests that missed the deadline TODO percentage?
	missed [2]int
	//Rolling average of init times when cold start
	initTime  [initPoints]float64
	initIndex int
}

var mut sync.Mutex

var m = make(map[string]functionInfo)

var rtt [rttPoints]float64
var rttIndex = 0

func Decide(r *scheduledRequest) int {
	name := r.Fun.Name
	fInfo, prs := m[name]

	//warmContainers, isWarm := node.WarmStatus()[name]

	//log.Printf("There are %t-%d containers for %s\n", isWarm, warmContainers, name)

	//Consider RTT?

	//warmContainers, isWarm := node.WarmStatus()[name]

	if prs && fInfo.meanResponse[0] > fInfo.meanResponse[1]+getRTT() && r.CanDoOffloading {
		return EXEC_CLOUD_REQUEST
	} else {
		return EXEC_LOCAL_REQUEST
	}
}

func InitDecisionEngine() {
	go ShowData()
}

func ShowData() {
	for {
		log.Println(m)
		log.Println(node.WarmStatus())
		log.Printf("Offload latency %f\n", getRTT())
		log.Printf("Init latency %f\n", getInitTime("sleep1"))
		log.Printf("Available CPU: %d Mem: %d MB", node.Resources.AvailableCPUs, node.Resources.AvailableMemMB)
		time.Sleep(5 * time.Second)
	}
}

func getRTT() float64 {
	sum := 0.0

	for i := 0; i < rttPoints; i++ {
		sum += rtt[i]
	}

	return sum / rttPoints
}

func getInitTime(name string) float64 {
	sum := 0.0
	x := initPoints

	//Rolling average not complete
	if m[name].count[1] < initPoints {
		x = m[name].count[1]
	}

	for i := 0; i < x; i++ {
		sum += m[name].initTime[i]
	}

	return sum / float64(x)
}

func Completed(r *function.Request, offloaded bool) {
	go updateData(r, offloaded)
}

// TODO sync or single thread?
func updateData(r *function.Request, offloaded bool) {
	name := r.Fun.Name
	var off int

	if offloaded {
		off = 1
	} else {
		off = 0
	}

	if offloaded {
		rtt[rttIndex] = r.ExecReport.OffloadLatency
		rttIndex++
	}

	mut.Lock()

	fInfo, prs := m[name]

	if !prs {
		fInfo = functionInfo{name: name}
	}

	if !r.ExecReport.IsWarmStart {
		fInfo.initTime[fInfo.initIndex] = r.ExecReport.InitTime
		fInfo.initIndex++
	}

	fInfo.count[off] = fInfo.count[off] + 1
	log.Printf("Completed %t-%d in %f deadline is %f\n", offloaded, off, r.ExecReport.ResponseTime, r.RequestQoS.MaxRespT)

	//One-pass mean and variance
	diff := r.ExecReport.ResponseTime - fInfo.meanResponse[off]
	fInfo.meanResponse[off] = fInfo.meanResponse[off] +
		(1/float64(fInfo.count[off]))*(diff)
	diff2 := r.ExecReport.ResponseTime - fInfo.meanResponse[off]

	fInfo.varianceResponse[off] = (diff * diff2) / float64(fInfo.count[off])

	if r.RequestQoS.MaxRespT != -1 && r.ExecReport.ResponseTime > r.RequestQoS.MaxRespT {
		log.Println("MISSED DEADLINE")
		fInfo.missed[off]++
	}

	m[name] = fInfo

	mut.Unlock()
}
