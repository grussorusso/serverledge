package scheduling

//
//import (
//	"github.com/grussorusso/serverledge/internal/function"
//	"github.com/grussorusso/serverledge/internal/node"
//	"log"
//	"math"
//	"reflect"
//	"strings"
//	"sync"
//	"time"
//)
//
///*
//const (
//	DROP_REQUEST           = 0
//	EXEC_LOCAL_REQUEST     = 1
//	EXEC_CLOUD_REQUEST     = 2
//	EXEC_NEIGHBOUR_REQUEST = 3
//)
//*/
//
//
//const (
//	LOCAL     = 0
//	CLOUD     = 1
//	NEIGHBOUR = 2
//)
//
//const rttPoints = 10
//const initPoints = 10
//
//type functionInfo_old struct {
//	name string
//	//Number of function requests
//	count [3]int
//	//Mean duration time
//	meanDuration [3]float64
//	//Variance of the duration time
//	varianceDuration [3]float64
//	//Number of requests that missed the deadline
//	missed [3]int
//	//Rolling average of init times when cold start
//	initTime  [3][initPoints]float64
//	initIndex [3]int
//}
//
//var mut_old sync.Mutex
//
//var m_old = make(map[string]functionInfo)
//
//var rtt [rttPoints]float64
//var rttIndex = 0
//
//const arrivalWindow = 5
//
//var arrivalList = createExpiringList(time.Second * arrivalWindow)
//var arrivalChannel = make(chan *scheduledRequest, 50)
//
//func Decide_old(r *scheduledRequest) (int, string) {
//	name := r.Fun.Name
//
//	arrivalChannel <- r
//
//	fInfo, prs := m[name]
//
//	//HIGH PERFORMANCE?
//	/*
//		if r.RequestQoS.Class == function.HIGH_PERFORMANCE {
//			return EXEC_CLOUD_REQUEST
//		}
//	*/
//
//	log.Printf("Prob LOCAL time greater than CLOUD is: %f\n",
//		normalCumulativeDistribution(
//			0,
//			fInfo.meanDuration[LOCAL]-(fInfo.meanDuration[CLOUD]+getRTT()),
//			fInfo.varianceDuration[LOCAL]+fInfo.varianceDuration[CLOUD]))
//	log.Printf("Utilizations:\n\tMem: %f\n\tCPU:%f\n",
//		float64(node.Resources.AvailableMemMB)/float64(node.Resources.MaxMemMB),
//		node.Resources.AvailableCPUs/node.Resources.MaxCPUs)
//	log.Printf("Warm servers in cloud %d\n", getWarmContainersInCloud(r))
//	log.Println(reflect.TypeOf(r.Params["n"]))
//
//	//If there isn't enough data execute locally
//	if !prs {
//		return EXEC_LOCAL_REQUEST, ""
//	}
//
//	//Return the URL as the output, maybe support multiple clouds
//	url, edgeRtt := getEdgeNodeOffloadingRtt(r)
//
//	log.Printf("Searching for edge offloading %f - %s\n", edgeRtt, url)
//
//	//If the QoS is too stringent to offload try to execute in local node TODO Is it correct?
//	if r.RequestQoS.MaxRespT != -1 && r.RequestQoS.MaxRespT < getRTT() {
//		r.CanDoOffloading = false
//		return EXEC_LOCAL_REQUEST, ""
//	}
//
//	_, isWarm := node.WarmStatus()[name]
//
//	//If there aren't enough resources for cold start offload
//	//TODO sync this operation?
//	if r.CanDoOffloading && !isWarm && node.Resources.AvailableCPUs < r.Fun.CPUDemand &&
//		node.Resources.AvailableMemMB < r.Fun.MemoryMB {
//		if url == "" ||
//			fInfo.meanDuration[NEIGHBOUR]+edgeRtt > fInfo.meanDuration[CLOUD]+getRTT()+getInitTime(name, CLOUD) {
//			return EXEC_CLOUD_REQUEST, ""
//		} else {
//			return EXEC_NEIGHBOUR_REQUEST, url
//		}
//	}
//
//	//If there are warm containers in the cloud might be advantageous to offload
//	if r.CanDoOffloading && !isWarm && getRTT() < getInitTime(name, LOCAL) {
//		return EXEC_CLOUD_REQUEST, ""
//	}
//
//	var coldStartTime float64
//	if isWarm {
//		coldStartTime = 0
//	} else {
//		coldStartTime = getInitTime(name, LOCAL)
//	}
//
//	//If the average duration is shorter in the cloud execute in the cloud
//	if r.CanDoOffloading &&
//		fInfo.meanDuration[LOCAL]+coldStartTime > fInfo.meanDuration[CLOUD]+getRTT()+getInitTime(name, CLOUD) &&
//		fInfo.missed[CLOUD] < fInfo.missed[LOCAL] {
//		return EXEC_CLOUD_REQUEST, ""
//	}
//
//	return EXEC_LOCAL_REQUEST, ""
//}
//
//// Probability (X LOCAL time, Y CLOUD time) P(X > Y) = P(X - Y > 0) = P(Z > 0)
//// mean = meanx - meany, var = varx + vary
//// X and Y are normally distributed?
//func normalCumulativeDistribution(x float64, mean float64, variance float64) float64 {
//	t := (x - mean) / (math.Sqrt(variance) * math.Sqrt2)
//	return 0.5 * (1 + math.Erf(t))
//}
//
//func InitDecisionEngine_old() {
//	//Debug
//	go ShowData()
//
//	go janitor()
//	go listHandler()
//}
//
//func listHandler() {
//	var r *scheduledRequest
//	for {
//		r = <-arrivalChannel
//		//r.Arrival Or time.now?
//		arrivalList.Add(r.Fun.Name, time.Now())
//	}
//}
//
//func janitor() {
//	for {
//		time.Sleep(5 * time.Second)
//
//		arrivalList.DeleteExpired(time.Now())
//	}
//}
//
//func ShowData_old() {
//	for {
//		time.Sleep(5 * time.Second)
//		log.Println("---------------------REPORT---------------------")
//		log.Println(m)
//		log.Println(node.WarmStatus())
//		log.Printf("Offload latency %f\n", getRTT())
//		log.Printf("Init latency %f\n", getInitTime("sleep1", LOCAL))
//		log.Println(arrivalList.GetList())
//		log.Println(node.Resources.String())
//		//log.Println(registration.Reg.CloudServersMap)
//	}
//}
//
//func getRTT() float64 {
//	sum := 0.0
//
//	for i := 0; i < rttPoints; i++ {
//		sum += rtt[i]
//	}
//
//	return sum / rttPoints
//}
//
//func getInitTime(name string, offload int) float64 {
//	sum := 0.0
//	x := initPoints
//
//	_, prs := m[name]
//	if !prs {
//		return -1
//	}
//
//	//Rolling average not complete
//	if m[name].count[offload] < initPoints {
//		x = m[name].count[offload]
//	}
//
//	for i := 0; i < x; i++ {
//		sum += m[name].initTime[offload][i]
//	}
//
//	return sum / float64(x)
//}
//
//// Completed TODO sync or single thread posting in a channel?
//func Completed_old(r *function.Request, offloaded int) {
//	go updateData(r, offloaded)
//}
//
//// Delete TODO modify API to call this function when a function is deleted
//func Delete_old(name string) {
//	delete(m, name)
//}
//
//// UpdateDataAsync TODO Use reqID to get missing information?
//func UpdateDataAsync_old(resp function.Response, reqId string) {
//	var off int
//
//	//TODO not really correct as offloading to neighbour is not considered
//	if resp.CloudOffloadLatency == 0 {
//		off = LOCAL
//	} else {
//		off = CLOUD
//	}
//
//	name := reqId[:strings.LastIndex(reqId, "-")]
//
//	mut.Lock()
//
//	if off == CLOUD {
//		rtt[rttIndex] = resp.CloudOffloadLatency
//		rttIndex = (rttIndex + 1) % rttPoints
//	}
//
//	fInfo, prs := m[name]
//
//	if !prs {
//		fInfo = functionInfo{name: name}
//	}
//
//	if !resp.IsWarmStart {
//		fInfo.initTime[off][fInfo.initIndex[off]] = resp.InitTime
//		fInfo.initIndex[off] = (fInfo.initIndex[off] + 1) % initPoints
//	}
//
//	fInfo.count[off] = fInfo.count[off] + 1
//
//	//Welford mean and variance
//	diff := resp.Duration - fInfo.meanDuration[off]
//	fInfo.meanDuration[off] = fInfo.meanDuration[off] +
//		(1/float64(fInfo.count[off]))*(diff)
//	diff2 := resp.Duration - fInfo.meanDuration[off]
//
//	fInfo.varianceDuration[off] = (diff * diff2) / float64(fInfo.count[off])
//
//	m[name] = fInfo
//
//	mut.Unlock()
//}
//
//func updateData_old(r *function.Request, location int) {
//	name := r.Fun.Name
//
//	mut.Lock()
//
//	if location != LOCAL {
//		rtt[rttIndex] = r.ExecReport.CloudOffloadLatency
//		rttIndex = (rttIndex + 1) % rttPoints
//	}
//
//	fInfo, prs := m[name]
//
//	if !prs {
//		fInfo = functionInfo{name: name}
//	}
//
//	if !r.ExecReport.IsWarmStart {
//		fInfo.initTime[location][fInfo.initIndex[location]] = r.ExecReport.InitTime
//		fInfo.initIndex[location] = (fInfo.initIndex[location] + 1) % initPoints
//	}
//
//	fInfo.count[location] = fInfo.count[location] + 1
//	log.Printf("Completed %d in %f deadline is %f\n", location, r.ExecReport.Duration, r.RequestQoS.MaxRespT)
//
//	//Welford mean and variance
//	diff := r.ExecReport.Duration - fInfo.meanDuration[location]
//	fInfo.meanDuration[location] = fInfo.meanDuration[location] +
//		(1/float64(fInfo.count[location]))*(diff)
//	diff2 := r.ExecReport.Duration - fInfo.meanDuration[location]
//
//	fInfo.varianceDuration[location] = (diff * diff2) / float64(fInfo.count[location])
//
//	if r.RequestQoS.MaxRespT != -1 && r.ExecReport.ResponseTime > r.RequestQoS.MaxRespT {
//		log.Println("MISSED DEADLINE")
//		fInfo.missed[location]++
//	}
//
//	m[name] = fInfo
//
//	mut.Unlock()
//}
//*/
