package scheduling

import (
	"context"
	"fmt"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"log"
	"math/rand"
	"time"
)

type metricGrabberFlux struct {
	m map[string]*functionInfo
}

var clientInflux influxdb2.Client
var writeAPI api.WriteAPI
var queryAPI api.QueryAPI
var deleteAPI api.DeleteAPI
var orgServerledge domain.Organization
var bucketServerledge *domain.Bucket

var bucketName string

func (g *metricGrabberFlux) InitMetricGrabber() {
	s := rand.NewSource(time.Now().UnixNano())
	rGen = rand.New(s)

	orgName := config.GetString(config.STORAGE_DB_ORGNAME, "serverledge")
	address := config.GetString(config.STORAGE_DB_ADDRESS, "http://localhost:8086")
	token := config.GetString(config.STORAGE_DB_TOKEN, "serverledge")

	log.Printf("Organization %s at %s\n", orgName, address)

	// TODO edit batch size
	// Get InfluxDB organization
	clientInflux = influxdb2.NewClientWithOptions(address, token,
		influxdb2.DefaultOptions().SetBatchSize(20))
	orgsAPI := clientInflux.OrganizationsAPI()
	bucketAPI := clientInflux.BucketsAPI()
	orgs, err := orgsAPI.GetOrganizations(context.Background(), api.PagingWithDescending(true))
	if err != nil {
		log.Fatal("Organization API error", err)
	}

	found := false
	for _, org := range *orgs {
		if orgName == org.Name {
			log.Printf("Found organization %s\n", org.Name)
			found = true
			orgServerledge = org
		}
	}

	var orgId string

	if !found {
		orgId = "serverledge"
		name := "Serverledge organization"
		timeNow := time.Now()
		_, err := orgsAPI.CreateOrganization(context.Background(), &domain.Organization{
			CreatedAt:   &timeNow,
			Description: &name,
			Id:          &orgId,
			Name:        orgName,
		})
		if err != nil {
			log.Fatal(err)
		}
	} else {
		orgId = *orgServerledge.Id
	}

	// Create a new bucket for the local node
	found = false
	bucketName = "serverledge-" + node.NodeIdentifier
	buckets, err := bucketAPI.GetBuckets(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for _, bucket := range *buckets {
		if bucketName == bucket.Name {
			log.Printf("Found bucket %s\n", bucket.Name)
			found = true
			bucketServerledge = &bucket
			break
		}
	}

	if !found {
		log.Printf("Creating bucket %s\n", bucketName)

		bucketServerledge, err = bucketAPI.CreateBucket(context.Background(), &domain.Bucket{
			Id:    &bucketName,
			Name:  bucketName,
			OrgID: &orgId,
		})

		if err != nil {
			log.Fatal(err)
		}
	}

	writeAPI = clientInflux.WriteAPI(orgName, bucketName)
	queryAPI = clientInflux.QueryAPI(orgName)
	deleteAPI = clientInflux.DeleteAPI()

	g.m = make(map[string]*functionInfo)

	evaluationInterval = time.Duration(config.GetInt(config.SOLVER_EVALUATION_INTERVAL, 10)) * time.Second
	// FIXME AUDIT log.Println("Evaluation interval:", evaluationInterval)

	// FIXME AUDIT go g.ShowData()
	go g.handler()
}

// FIXME set audit
func (g *metricGrabberFlux) ShowData() {
	for {
		time.Sleep(time.Second * 10)
		for _, fInfo := range g.m {
			for _, cFInfo := range fInfo.invokingClasses {
				log.Println(cFInfo)
			}
		}
	}
}

/*
Function that:
- Handles the evaluation and calculation of the local, edge and cloud probabilities.
- Writes the report of the request completion into the data store (influxdb).
- With the arrival of a new request, initializes new functionInfo and classFunctionInfo objects.
*/
func (g *metricGrabberFlux) handler() {
	evaluationTicker :=
		time.NewTicker(evaluationInterval)
	for {
		select {
		case _ = <-evaluationTicker.C: // Evaluation handler
			s := rand.NewSource(time.Now().UnixNano())
			rGen = rand.New(s)
			//log.Println("Evaluating")

			//Check if there are some instances with 0 arrivals
			for fName, fInfo := range g.m {
				for cName, cFInfo := range fInfo.invokingClasses {
					//Cleanup
					if cFInfo.arrivalCount == 0 {
						cFInfo.timeSlotsWithoutArrivals++
						if cFInfo.timeSlotsWithoutArrivals >= maxTimeSlots {
							log.Println("DELETING", fName, cName)
							g.Delete(fName, cName)
						}
					}
				}
			}

			//d.deleteOldData(24 * time.Hour)
			g.queryMetrics()
			g.updateProbabilities()

		case r := <-requestChannel: // Result storage handler
			// New request completed or dropped - added data to influxdb - need to differentiate between edge offloading and cloud offloading
			// completed: true if the completed request is not dropped
			// offloaded: true if the request is offloaded to another node
			// offloaded_cloud: true if the completed request is offloaded vertically
			// offloaded_edge: true if the completed request is offloaded horizontally
			// warm_start: true if there were available instances to execute the function locally
			// fKeys: contains extra information about the function execution
			// - duration
			// - init_time
			// FIXME AUDIT log.Println("Result storage handler - adding data to influxdb")

			var fKeys map[string]interface{}
			offloaded := "false"
			offloadedCloud := "false"
			warmStart := "false"
			completed := "false"

			// If the request was dropped, then update the respective value in the node structure
			if r.dropped {
				node.Resources.DropRequestsCount += 1
				fKeys = map[string]interface{}{
					"duration":   r.ExecReport.Duration,
					"init_time":  r.ExecReport.InitTime,
					"input_size": r.ExecReport.InputSize,
				}
				completed = "false"
			} else {
				fKeys = map[string]interface{}{
					"duration":   r.ExecReport.Duration,
					"init_time":  r.ExecReport.InitTime,
					"input_size": r.ExecReport.InputSize,
				}
				completed = "true"
			}

			if r.ExecReport.OffloadLatencyCloud != 0 {
				offloaded = "true"
				offloadedCloud = "true"
				fKeys["offload_latency_cloud"] = r.ExecReport.OffloadLatencyCloud
			}

			if r.ExecReport.OffloadLatencyEdge != 0 {
				offloaded = "true"
				offloadedCloud = "false"
				fKeys["offload_latency_edge"] = r.ExecReport.OffloadLatencyEdge
			}

			if r.ExecReport.IsWarmStart {
				warmStart = "true"
			}

			p := influxdb2.NewPoint(r.Fun.Name,
				map[string]string{
					"class":           r.ClassService.Name,
					"offloaded":       offloaded,
					"offloaded_cloud": offloadedCloud,
					"warm_start":      warmStart,
					"completed":       completed},
				fKeys,
				time.Now())

			writeAPI.WritePoint(p)
			// FIXME AUDIT log.Println("ADDED NEW POINT INTO INFLUXDB")

		case arr := <-arrivalChannel: // Arrival handler - structures initialization
			// A new request is arrived: update the counter of incoming request in the node structure
			// FIXME AUDIT log.Println("NEW ARRIVAL!")
			node.Resources.RequestsCount += 1

			name := arr.Fun.Name
			fInfo, prs := g.m[name]
			if !prs {
				fInfo = &functionInfo{
					name:            name,
					memory:          arr.Fun.MemoryMB,
					cpu:             arr.Fun.CPUDemand,
					probCold:        [3]float64{0, 0, 0},
					bandwidthCloud:  0,
					bandwidthEdge:   0,
					meanInputSize:   100,
					invokingClasses: make(map[string]*classFunctionInfo)}

				g.m[name] = fInfo
			}

			cFInfo, prs := fInfo.invokingClasses[arr.class]
			if !prs {
				cFInfo = &classFunctionInfo{functionInfo: fInfo,
					probExecuteLocal:         startingLocalProb,
					probOffloadCloud:         startingCloudOffloadProb,
					probOffloadEdge:          startingEdgeOffloadProb,
					probDrop:                 1 - (startingLocalProb + startingCloudOffloadProb + startingEdgeOffloadProb),
					arrivals:                 0,
					arrivalCount:             0,
					timeSlotsWithoutArrivals: 0,
					className:                arr.class}

				fInfo.invokingClasses[arr.class] = cFInfo
			}

			cFInfo.timeSlotsWithoutArrivals = 0

		}
	}
}

func (g *metricGrabberFlux) queryMetrics() {
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

func (g *metricGrabberFlux) GrabFunctionInfo(functionName string) (*functionInfo, bool) {
	fInfo, prs := g.m[functionName]
	if !prs {
		log.Printf("Function with name %s is not present in cache.", functionName)
	}
	return fInfo, prs
}

func (g *metricGrabberFlux) updateProbabilities() {
	solve(g.m)
}

// Completed : this method is executed only in case the request is not dropped and
// takes in input a 'scheduledRequest' object and an integer 'offloaded' that can have 3 possible values:
// 1) offloaded = LOCAL = 0 --> the request is executed locally and not offloaded
// 2) offloaded = OFFLOADED_CLOUD = 1 --> the request is offloaded to cloud
// 3) offloaded = OFFLOADED_EDGE = 2 --> the request is offloaded to edge node
// Triggers the DB update
func (g *metricGrabberFlux) Completed(r *scheduledRequest, offloaded int) {
	/*if offloaded == 0 {
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

func (g *metricGrabberFlux) Delete(function string, class string) {
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
