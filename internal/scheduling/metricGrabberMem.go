package scheduling

import (
	"context"
	"fmt"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
	"log"
	"math/rand"
	"time"
)

type metricGrabberMem struct {
	m map[string]*functionInfo
}

func (g *metricGrabberMem) InitMetricGrabber() {
	s := rand.NewSource(time.Now().UnixNano())
	rGen = rand.New(s)

	g.m = make(map[string]*functionInfo)

	evaluationInterval = time.Duration(config.GetInt(config.SOLVER_EVALUATION_INTERVAL, 10)) * time.Second
	log.Println("Evaluation interval:", evaluationInterval)
	go g.handler()
}

func (g *metricGrabberMem) ShowData() {
	for {
		time.Sleep(time.Second * 10)
		for _, fInfo := range g.m {
			for _, cFInfo := range fInfo.invokingClasses {
				log.Println(cFInfo)
			}
		}
	}
	//log.Println("ERLANG: ", ErlangB(57, 45))
	//for {
	//	time.Sleep(5 * time.Second)
	//	log.Println("map", m)
	//}
	/*
		for {
			time.Sleep(5 * time.Second)
			for _, functionMap := range m {
				for _, finfo := range functionMap {
					log.Println(finfo)
				}
			}
		}
	*/
}

/*
Function that:
- Handles the evaluation and calculation of the cold start probabilities.
- Writes the report of the request completion into the data store (influxdb).
- With the arrival of a new request, initializes new functionInfo and classFunctionInfo objects.
*/
func (g *metricGrabberMem) handler() {
	evaluationTicker :=
		time.NewTicker(evaluationInterval)
	pcoldTicker :=
		time.NewTicker(time.Duration(config.GetInt(config.CONTAINER_EXPIRATION_TIME, 600)) * time.Second)

	for {
		select {
		case _ = <-evaluationTicker.C: // Evaluation handler
			s := rand.NewSource(time.Now().UnixNano())
			rGen = rand.New(s)
			log.Println("Evaluating")

			//Check if there are some instances with 0 arrivals
			for fName, fInfo := range g.m {
				for cName, cFInfo := range fInfo.invokingClasses {
					//Cleanup
					if cFInfo.arrivalCount == 0 {
						cFInfo.timeSlotsWithoutArrivals++
						if cFInfo.timeSlotsWithoutArrivals >= maxTimeSlots {
							g.Delete(fName, cName)
						}
					}
				}
			}

			g.updateProbabilities()

			//Reset arrivals for the time slot
			for _, fInfo := range g.m {
				for _, cFInfo := range fInfo.invokingClasses {
					//Cleanup
					cFInfo.arrivalCount = 0
					cFInfo.arrivals = 0
				}
			}

		case r := <-requestChannel: // Result storage handler
			// New request completed - data is updated in local memory - need to differentiate between edge offloading and cloud offloading
			// Also need to increment the number of blocked requests in the node if this is the case

			// If the request was dropped, then update the respective value in the node structure
			if r.dropped {
				node.Resources.DropRequestsCount += 1
			}

			g.updateData(r)
		case arr := <-arrivalChannel: // Arrival handler - structures initialization
			// A new request is arrived: update the counter of incoming request in the node structure
			node.Resources.RequestsCount += 1

			name := arr.Fun.Name

			fInfo, prs := g.m[name]
			if !prs {
				fInfo = &functionInfo{
					name:            name,
					memory:          arr.Fun.MemoryMB,
					cpu:             arr.Fun.CPUDemand,
					probCold:        [3]float64{1, 1, 1},
					bandwidthCloud:  0,
					bandwidthEdge:   0,
					invokingClasses: make(map[string]*classFunctionInfo)}

				g.m[name] = fInfo
			}

			cFInfo, prs := fInfo.invokingClasses[arr.class]
			if !prs {
				cFInfo = &classFunctionInfo{functionInfo: fInfo,
					probExecuteLocal:         startingLocalProb,
					probOffloadCloud:         startingCloudOffloadProb,
					probOffloadEdge:          startingEdgeOffloadProb,
					probDrop:                 1 - (startingLocalProb + startingCloudOffloadProb),
					arrivals:                 0,
					arrivalCount:             0,
					timeSlotsWithoutArrivals: 0,
					className:                arr.class}

				fInfo.invokingClasses[arr.class] = cFInfo
			}

			cFInfo.arrivalCount++
			cFInfo.arrivals = cFInfo.arrivalCount / float64(evaluationInterval)
			cFInfo.timeSlotsWithoutArrivals = 0

		case _ = <-pcoldTicker.C:
			//Reset arrivals for the time slot
			for _, fInfo := range g.m {
				fInfo.coldStartCount = [3]int64{0, 0, 0}
				fInfo.timeSlotCount = [3]int64{0, 0, 0}
			}
		}
	}
}

func (g *metricGrabberMem) queryMetrics() {
	//TODO edit time window
	searchInterval := 24 * time.Hour

	//Query for arrivals
	for _, fInfo := range g.m {
		for _, cFInfo := range fInfo.invokingClasses {
			cFInfo.arrivals = 0
		}
	}

	start := time.Now().Add(-evaluationInterval)
	query := fmt.Sprintf(`from(bucket: "%s")
										|> range(start: %d)
										|> filter(fn: (r) => r["_field"] == "duration")
										|> group(columns: ["_measurement", "class"])
									    |> aggregateWindow(every: 1s, fn: count, createEmpty: true)
									    |> mean()`, bucketName, start.Unix())

	result, err := queryAPI.Query(context.Background(), query)
	if err == nil {
		// Iterate over query response
		for result.Next() {
			x := result.Record().Values()
			val := result.Record().Value().(float64)
			funct := x["_measurement"].(string)
			class := x["class"].(string)

			fInfo, prs := g.m[funct] // access function map in Decision Engine
			if !prs {
				f, _ := function.GetFunction(funct)
				fInfo = &functionInfo{
					name:            funct,
					memory:          f.MemoryMB,
					cpu:             f.CPUDemand,
					probCold:        [3]float64{0, 0, 0},
					invokingClasses: make(map[string]*classFunctionInfo)}

				g.m[funct] = fInfo
			}

			//timeWindow := 25 * 60.0
			cFInfo, prs := fInfo.invokingClasses[class]
			if !prs {
				cFInfo = &classFunctionInfo{functionInfo: fInfo,
					probExecuteLocal:         startingLocalProb,
					probOffloadCloud:         startingCloudOffloadProb,
					probDrop:                 1 - (startingLocalProb + startingCloudOffloadProb),
					arrivals:                 0,
					arrivalCount:             0,
					timeSlotsWithoutArrivals: 0,
					className:                class}

				fInfo.invokingClasses[class] = cFInfo
			}
			cFInfo.arrivals = val
			// FIXME REMOVE log.Println("Recovered arrivals from influxDb: ", cFInfo.arrivals)

			//Reset deletion
			cFInfo.timeSlotsWithoutArrivals = 0
		}

		// check for an error
		if result.Err() != nil {
			log.Printf("query parsing error: %s\n", result.Err().Error())
		}
	} else {
		log.Println("DB error", err)
	}

	// Query for meanDuration
	start = time.Now().Add(-searchInterval)
	query = fmt.Sprintf(`from(bucket: "%s")
										|> range(start: %d)
										|> group(columns: ["_measurement", "offloaded", "offloaded_cloud"])
										|> filter(fn: (r) => r["_field"] == "duration" and r["completed"] == "true")
										|> tail(n: %d)
										|> exponentialMovingAverage(n: %d)`, bucketName, start.Unix(), 100, 100)

	result, err = queryAPI.Query(context.Background(), query)
	if err == nil {
		// Iterate over query response
		for result.Next() {
			x := result.Record().Values()
			val := result.Record().Value().(float64)

			funct := x["_measurement"].(string)
			off := x["offloaded"].(string)
			offCloud := x["offloaded_cloud"].(string)

			// retrieve location value to check if the function was executed locally, on cloud or on edge
			location := LOCAL
			if off == "true" && offCloud == "true" {
				location = OFFLOADED_CLOUD
			} else if off == "true" && offCloud == "false" {
				location = OFFLOADED_EDGE
			}
			fInfo, prs := g.m[funct]
			if !prs {
				continue
			}

			fInfo.meanDuration[location] = val
		}

		// check for an error
		if result.Err() != nil {
			log.Printf("query parsing error: %s\n", result.Err().Error())
		}
	} else {
		log.Println("DB error", err)
	}

	// Query for OffloadLatencyCloud
	query = fmt.Sprintf(`from(bucket: "%s")
										|> range(start: %d)
										|> filter(fn: (r) => r["_field"] == "offload_latency_cloud" and r["completed"] == "true")
										|> group()
										|> median()`, bucketName, start.Unix())

	result, err = queryAPI.Query(context.Background(), query)
	if err == nil {
		// Iterate over query response
		for result.Next() {
			CloudOffloadLatency = result.Record().Values()["_value"].(float64)
		}

		// check for an error
		if result.Err() != nil {
			log.Printf("query parsing error: %s\n", result.Err().Error())
		}
	} else {
		log.Println("DB error", err)
	}

	// Query for offloadLatencyEdge
	query = fmt.Sprintf(`from(bucket: "%s")
										|> range(start: %d)
										|> filter(fn: (r) => r["_field"] == "offload_latency_edge" and r["completed"] == "true")
										|> group()
										|> median()`, bucketName, start.Unix())

	result, err = queryAPI.Query(context.Background(), query)
	if err == nil {
		// Iterate over query response
		for result.Next() {
			EdgeOffloadLatency = result.Record().Values()["_value"].(float64)
		}

		// check for an error
		if result.Err() != nil {
			log.Printf("query parsing error: %s\n", result.Err().Error())
		}
	} else {
		log.Println("DB error", err)
	}

	//Query for initTime
	query = fmt.Sprintf(`from(bucket: "%s")
										|> range(start: %d)
										|> group(columns: ["_measurement", "offloaded", "offloaded_cloud"])
										|> filter(fn: (r) => r["_field"] == "init_time" and r["warm_start"] == "false" and r["completed"] == "true")
										|> tail(n: %d)
										|> exponentialMovingAverage(n: %d)`, bucketName, start.Unix(), 100, 100)
	result, err = queryAPI.Query(context.Background(), query)
	if err == nil {
		// Iterate over query response
		for result.Next() {
			x := result.Record().Values()
			val := result.Record().Value().(float64)

			funct := x["_measurement"].(string)
			off := x["offloaded"].(string)
			offCloud := x["offloaded_cloud"].(string)

			location := LOCAL
			if off == "true" && offCloud == "true" {
				location = OFFLOADED_CLOUD
			} else if off == "true" && offCloud == "false" {
				location = OFFLOADED_EDGE
			}

			fInfo, prs := g.m[funct]
			if !prs {
				continue
			}

			fInfo.initTime[location] = val
		}

		// check for an error
		if result.Err() != nil {
			log.Printf("query parsing error: %s\n", result.Err().Error())
		}
	} else {
		log.Println("DB error", err)
	}

	// Query for input size
	query = fmt.Sprintf(`from(bucket: "%s")
										|> range(start: %d)
										|> group(columns: ["_measurement"])
										|> filter(fn: (r) => r["_field"] == "input_size" and r["completed"] == "true")
										|> tail(n: %d)
										|> exponentialMovingAverage(n: %d)`, bucketName, start.Unix(), 100, 100)
	result, err = queryAPI.Query(context.Background(), query)
	if err == nil {
		// Iterate over query response
		for result.Next() {
			x := result.Record().Values()
			val := result.Record().Value().(float64)

			funct := x["_measurement"].(string)
			fInfo, prs := g.m[funct]
			if !prs {
				continue
			}
			fInfo.meanInputSize = val
		}

		// check for an error
		if result.Err() != nil {
			log.Printf("query parsing error: %s\n", result.Err().Error())
		}
	} else {
		log.Println("DB error", err)
	}

	// Query for count and coldStartCount
	query = fmt.Sprintf(`from(bucket: "%s")
										|> range(start: %d)
  										|> filter(fn: (r) => r["_field"] == "duration" and r["completed"] == "true")
										|> group(columns: ["_measurement", "offloaded", "offloaded_cloud", "warm_start"])
										|> count()`, bucketName, start.Unix())

	result, err = queryAPI.Query(context.Background(), query)
	if err == nil {
		// Iterate over query response
		for result.Next() {
			x := result.Record().Values()
			val := result.Record().Value().(int64)

			funct := x["_measurement"].(string)
			off := x["offloaded"].(string)
			offCloud := x["offloaded_cloud"].(string)
			warmStart := x["warm_start"].(string)

			location := LOCAL
			if off == "true" && offCloud == "true" {
				location = OFFLOADED_CLOUD
			} else if off == "true" && offCloud == "false" {
				location = OFFLOADED_EDGE
			}

			fInfo, prs := g.m[funct]
			if !prs {
				continue
			}

			if warmStart == "true" {
				fInfo.count[location] = val
			} else {
				fInfo.coldStartCount[location] = val
			}
		}

		// check for an error
		if result.Err() != nil {
			log.Printf("query parsing error: %s\n", result.Err().Error())
		}
	} else {
		log.Println("DB error", err)
	}

	for _, fInfo := range g.m {
		// If none cold start happened in a specific location (local, cloud or edge), then the cold start probability is optimistically 0
		for location := 0; location < 3; location++ {
			if fInfo.coldStartCount[location] == 0 {
				fInfo.probCold[location] = 0.0
			} else {
				fInfo.probCold[location] = float64(fInfo.coldStartCount[location]) / float64(fInfo.count[location]+fInfo.coldStartCount[location])
			}
		}
	}
}

func (g *metricGrabberMem) GrabFunctionInfo(functionName string) (*functionInfo, bool) {
	fInfo, prs := g.m[functionName]
	if !prs {
		log.Printf("Function with name %s is not present in cache.", functionName)
	}
	return fInfo, prs
}

func (g *metricGrabberMem) updateProbabilities() {
	solve(g.m)
}

func (g *metricGrabberMem) updateData(r completedRequest) {
	name := r.Fun.Name

	location := r.location

	fInfo, prs := g.m[name]
	//TODO create here the entry in the function? Is it necessary?
	if !prs {
		return
	}

	fInfo.count[location] = fInfo.count[location] + 1
	fInfo.timeSlotCount[location] = fInfo.timeSlotCount[location] + 1

	//Welford mean and variance
	diff := r.ExecReport.Duration - fInfo.meanDuration[location]
	fInfo.meanDuration[location] = fInfo.meanDuration[location] +
		(1/float64(fInfo.count[location]))*(diff)
	diff2 := r.ExecReport.Duration - fInfo.meanDuration[location]

	fInfo.varianceDuration[location] = (diff * diff2) / float64(fInfo.count[location])

	if !r.ExecReport.IsWarmStart {
		diff := r.ExecReport.InitTime - fInfo.initTime[location]
		fInfo.initTime[location] = fInfo.initTime[location] +
			(1/float64(fInfo.count[location]))*(diff)

		fInfo.coldStartCount[location]++
		fInfo.probCold[location] = float64(fInfo.coldStartCount[location]) / float64(fInfo.timeSlotCount[location])
	}

	// Update offload latency cloud
	if r.ExecReport.OffloadLatencyCloud != 0 {
		diff := r.ExecReport.OffloadLatencyCloud - CloudOffloadLatency
		CloudOffloadLatency = CloudOffloadLatency + (1/float64(fInfo.count[location]))*(diff)
	}

	// Update offload latency edge
	if r.ExecReport.OffloadLatencyEdge != 0 {
		diff := r.ExecReport.OffloadLatencyEdge - EdgeOffloadLatency
		EdgeOffloadLatency = EdgeOffloadLatency + (1/float64(fInfo.count[location]))*(diff)
	}

}

// Completed : this method is executed only in case the request is not dropped and
// takes in input a 'scheduledRequest' object and an integer 'offloaded' that can have 3 possible values:
// 1) offloaded = LOCAL = 0 --> the request is executed locally and not offloaded
// 2) offloaded = OFFLOADED_CLOUD = 1 --> the request is offloaded to cloud
// 3) offloaded = OFFLOADED_EDGE = 2 --> the request is offloaded to edge node
// Triggers the DB update
func (g *metricGrabberMem) Completed(r *scheduledRequest, offloaded int) {
	/*FIXME AUDIT if offloaded == 0 {
		log.Printf("LOCAL RESULT %s - Duration: %f, InitTime: %f", r.Fun.Name, r.ExecReport.Duration, r.ExecReport.InitTime)
	} else if offloaded == 1 {
		log.Printf("VERTICAL OFFLOADING RESULT %s - Duration: %f, InitTime: %f", r.Fun.Name, r.ExecReport.Duration, r.ExecReport.InitTime)
	} else {
		log.Printf("HORIZONTAL OFFLOADING RESULT %s - Duration: %f, InitTime: %f", r.Fun.Name, r.ExecReport.Duration, r.ExecReport.InitTime)
	}*/

	requestChannel <- completedRequest{
		scheduledRequest: r,
		location:         offloaded,
		dropped:          false,
	}
}

func (g *metricGrabberMem) Delete(function string, class string) {
	fInfo, prs := g.m[function]
	if !prs {
		return
	}

	delete(fInfo.invokingClasses, class)

	//If there aren't any more classes calls the function can be deleted
	if len(fInfo.invokingClasses) == 0 {
		delete(g.m, function)
	}
}
