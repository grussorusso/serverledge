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

const (
	LOCAL = 0
	CLOUD = 1
)

const rttPoints = 10
const initPoints = 10

type functionInfo struct {
	name string
	//Number of function requests
	count [2]int
	//Mean duration time
	meanDuration [2]float64
	//Variance of the duration time
	varianceDuration [2]float64
	//Number of requests that missed the deadline TODO percentage?
	missed [2]int
	//Rolling average of init times when cold start
	initTime  [2][initPoints]float64
	initIndex [2]int
}

var mut sync.Mutex

var m = make(map[string]functionInfo)

var rtt [rttPoints]float64
var rttIndex = 0

const arrivalWindow = 5

var arrivalList = createExpiringList(time.Second * arrivalWindow)
var arrivalChannel = make(chan *scheduledRequest, 10)

func Decide(r *scheduledRequest) int {
	name := r.Fun.Name

	arrivalChannel <- r

	fInfo, prs := m[name]

	//If there isn't enough data execute locally
	if !prs {
		return EXEC_LOCAL_REQUEST
	}

	//If the QoS is too stringent to offload try to execute in local node TODO Is it correct?
	if r.RequestQoS.MaxRespT != -1 && r.RequestQoS.MaxRespT < getRTT() {
		r.CanDoOffloading = false
		return EXEC_LOCAL_REQUEST
	}

	_, isWarm := node.WarmStatus()[name]

	//If there aren't enough resources for cold start execute in the cloud
	//TODO sync this operation?
	if r.CanDoOffloading && !isWarm && node.Resources.AvailableCPUs < r.Fun.CPUDemand &&
		node.Resources.AvailableMemMB < r.Fun.MemoryMB {
		return EXEC_CLOUD_REQUEST
	}

	var coldStartTime float64
	if isWarm {
		coldStartTime = 0
	} else {
		coldStartTime = getInitTime(name, LOCAL)
	}

	//If the average duration is shorter in the cloud execute in the cloud
	if r.CanDoOffloading &&
		fInfo.meanDuration[LOCAL]+coldStartTime > fInfo.meanDuration[CLOUD]+getRTT()+getInitTime(name, CLOUD) &&
		fInfo.missed[CLOUD] < fInfo.missed[LOCAL] {
		return EXEC_CLOUD_REQUEST
	}

	return EXEC_LOCAL_REQUEST
}

func InitDecisionEngine() {
	go ShowData()
	go janitor()
	go listHandler()
}

func listHandler() {
	var r *scheduledRequest
	for {
		r = <-arrivalChannel
		arrivalList.Add(r.Fun.Name, r.Arrival)
	}
}

func janitor() {
	for {
		time.Sleep(5 * time.Second)

		arrivalList.DeleteExpired(time.Now())
	}
}

func ShowData() {
	for {
		log.Println(m)
		log.Println(node.WarmStatus())
		log.Printf("Offload latency %f\n", getRTT())
		log.Printf("Init latency %f\n", getInitTime("sleep1", LOCAL))
		log.Println(arrivalList.GetList())
		log.Printf("Available CPU: %f Mem: %d MB", node.Resources.AvailableCPUs, node.Resources.AvailableMemMB)
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

func getInitTime(name string, offload int) float64 {
	sum := 0.0
	x := initPoints

	//Rolling average not complete
	if m[name].count[offload] < initPoints {
		x = m[name].count[offload]
	}

	for i := 0; i < x; i++ {
		sum += m[name].initTime[offload][i]
	}

	return sum / float64(x)
}

// Completed TODO sync or single thread posting in a channel?
func Completed(r *function.Request, offloaded bool) {
	go updateData(r, offloaded)
}

// Delete TODO modify API to call this function when a function is deleted
func Delete(name string) {
	delete(m, name)
}

func updateData(r *function.Request, offloaded bool) {
	name := r.Fun.Name
	var off int

	if offloaded {
		off = CLOUD
	} else {
		off = LOCAL
	}

	if offloaded {
		rtt[rttIndex] = r.ExecReport.OffloadLatency
		rttIndex = (rttIndex + 1) % rttPoints
	}

	mut.Lock()

	fInfo, prs := m[name]

	if !prs {
		fInfo = functionInfo{name: name}
	}

	if !r.ExecReport.IsWarmStart {
		fInfo.initTime[off][fInfo.initIndex[off]] = r.ExecReport.InitTime
		fInfo.initIndex[off] = (fInfo.initIndex[off] + 1) % initPoints
	}

	fInfo.count[off] = fInfo.count[off] + 1
	log.Printf("Completed %t-%d in %f deadline is %f\n", offloaded, off, r.ExecReport.Duration, r.RequestQoS.MaxRespT)

	//One-pass mean and variance
	diff := r.ExecReport.Duration - fInfo.meanDuration[off]
	fInfo.meanDuration[off] = fInfo.meanDuration[off] +
		(1/float64(fInfo.count[off]))*(diff)
	diff2 := r.ExecReport.Duration - fInfo.meanDuration[off]

	fInfo.varianceDuration[off] = (diff * diff2) / float64(fInfo.count[off])

	if r.RequestQoS.MaxRespT != -1 && r.ExecReport.ResponseTime > r.RequestQoS.MaxRespT {
		log.Println("MISSED DEADLINE")
		fInfo.missed[off]++
	}

	m[name] = fInfo

	mut.Unlock()
}
