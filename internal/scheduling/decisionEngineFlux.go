package scheduling

import (
	"context"
	"fmt"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"log"
	"math/rand"
	"time"
)

type decisionEngineFlux struct {
	m map[string]*functionInfo
}

var clientInflux influxdb2.Client
var writeAPI api.WriteAPI
var queryAPI api.QueryAPI

var bucketName string

func (d *decisionEngineFlux) Decide(r *scheduledRequest) int {
	name := r.Fun.Name
	class := r.ClassService

	prob := rGen.Float64()

	var pe float64
	var po float64
	var pd float64

	var cFInfo *classFunctionInfo

	arrivalChannel <- arrivalRequest{r, class.Name}

	fInfo, prs := d.m[name]
	if !prs {
		pe = startingExecuteProb
		po = startingOffloadProb
		pd = 1 - (pe + po)
	} else {
		cFInfo, prs = fInfo.invokingClasses[class.Name]
		if !prs {
			pe = startingExecuteProb
			po = startingOffloadProb
			pd = 1 - (pe + po)
		} else {
			pe = cFInfo.probExecute
			po = cFInfo.probOffload
			pd = cFInfo.probDrop
		}
	}

	log.Printf("Probabilities for %s-%s are %f %f %f", name, class.Name, pe, po, pd)

	//warmNumber, isWarm := node.WarmStatus()[name]
	if !r.CanDoOffloading {
		pd = pd / (pd + pe)
		pe = pe / (pd + pe)
		po = 0
	} else if node.Resources.AvailableCPUs < r.Fun.CPUDemand &&
		node.Resources.AvailableMemMB < r.Fun.MemoryMB {
		pd = pd / (pd + po)
		po = po / (pd + po)
		pe = 0
	}

	if prob <= pe {
		log.Println("Execute LOCAL")
		return EXECUTE_REQUEST
	} else if prob <= pe+po {
		log.Println("Execute OFFLOAD")
		return OFFLOAD_REQUEST
	} else {
		log.Println("Execute DROP")
		requestChannel <- completedRequest{
			scheduledRequest: r,
			dropped:          false,
		}

		return DROP_REQUEST
	}
}

func (d *decisionEngineFlux) InitDecisionEngine() {
	s := rand.NewSource(time.Now().UnixNano())
	rGen = rand.New(s)

	orgName := config.GetString(config.STORAGE_DB_ORGNAME, "serverledge")
	address := config.GetString(config.STORAGE_DB_ADDRESS, "http://localhost:8086")
	token := config.GetString(config.STORAGE_DB_TOKEN, "serverledge")

	log.Printf("Organization %s at %s\n", orgName, address)

	// TODO edit batch size
	clientInflux = influxdb2.NewClientWithOptions(address, token,
		influxdb2.DefaultOptions().SetBatchSize(20))
	orgsAPI := clientInflux.OrganizationsAPI()
	bucketAPI := clientInflux.BucketsAPI()
	orgs, err := orgsAPI.GetOrganizations(context.Background(), api.PagingWithDescending(true))
	if err != nil {
		log.Fatal("Organization API error", err)
	}

	found := false
	var orgServerledge domain.Organization
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
		}
	}

	if !found {
		log.Printf("Creating bucket %s\n", bucketName)
		_, err = bucketAPI.CreateBucket(context.Background(), &domain.Bucket{
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

	evaluationInterval = time.Duration(config.GetInt(config.SOLVER_EVALUATION_INTERVAL, 10)) * time.Second

	d.m = make(map[string]*functionInfo)

	go d.ShowData()
	go d.handler()
}

func (d *decisionEngineFlux) queryDb() {
	//TODO edit time window
	/*
			from(bucket: "completions")
			  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
			  |> group(columns: ["_measurement", "class"])
			  |> aggregateWindow(every: v.windowPeriod, fn: count, createEmpty: true)

		maybe |> max()
		or 	  |> mean()

			from(bucket: "completions")
											|> range(start: -%dm)
											|> group(columns: ["_measurement", "class"])
											|> count()
	*/
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

			fInfo, prs := d.m[funct]
			if !prs {
				f, _ := function.GetFunction(funct)
				fInfo = &functionInfo{
					name:            funct,
					memory:          f.MemoryMB,
					cpu:             f.CPUDemand,
					probCold:        [2]float64{1, 1},
					invokingClasses: make(map[string]*classFunctionInfo)}

				d.m[funct] = fInfo
			}

			//timeWindow := 25 * 60.0
			cFInfo, prs := fInfo.invokingClasses[class]
			if !prs {
				cFInfo = &classFunctionInfo{functionInfo: fInfo,
					probExecute:              startingExecuteProb,
					probOffload:              startingOffloadProb,
					probDrop:                 1 - (startingExecuteProb + startingOffloadProb),
					arrivals:                 0,
					arrivalCount:             0,
					timeSlotsWithoutArrivals: 0,
					className:                class}

				fInfo.invokingClasses[class] = cFInfo
			}
			cFInfo.arrivals = val

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

	query = fmt.Sprintf(`from(bucket: "%s")
										|> range(start: -%dh)
										|> group(columns: ["_measurement", "offloaded"])
										|> filter(fn: (r) => r["_field"] == "duration" and r["completed"] == "true")
										|> tail(n: %d)
										|> exponentialMovingAverage(n: %d)`, bucketName, 12, 100, 100)

	result, err = queryAPI.Query(context.Background(), query)
	if err == nil {
		// Iterate over query response
		for result.Next() {
			x := result.Record().Values()
			val := result.Record().Value().(float64)

			funct := x["_measurement"].(string)
			off := x["offloaded"].(string)
			location := LOCAL
			if off == "true" {
				location = OFFLOADED
			}
			fInfo, prs := d.m[funct]
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

	query = fmt.Sprintf(`from(bucket: "%s")
										|> range(start: -%dh)
										|> filter(fn: (r) => r["_field"] == "offload_latency" and r["completed"] == "true")
										|> group()
										|> tail(n: %d)
										|> exponentialMovingAverage(n: %d)`, bucketName, 12, 100, 100)

	result, err = queryAPI.Query(context.Background(), query)
	if err == nil {
		// Iterate over query response
		for result.Next() {
			OffloadLatency = result.Record().Values()["_value"].(float64)
		}

		// check for an error
		if result.Err() != nil {
			log.Printf("query parsing error: %s\n", result.Err().Error())
		}
	} else {
		log.Println("DB error", err)
	}

	query = fmt.Sprintf(`from(bucket: "%s")
										|> range(start: -%dh)
										|> group(columns: ["_measurement", "offloaded"])
										|> filter(fn: (r) => r["_field"] == "init_time" and r["warm_start"] == "false" and r["completed"] == "true")
										|> tail(n: %d)
										|> exponentialMovingAverage(n: %d)`, bucketName, 12, 100, 100)
	result, err = queryAPI.Query(context.Background(), query)
	if err == nil {
		// Iterate over query response
		for result.Next() {
			x := result.Record().Values()
			val := result.Record().Value().(float64)

			funct := x["_measurement"].(string)
			off := x["offloaded"].(string)

			location := LOCAL
			if off == "true" {
				location = OFFLOADED
			}

			fInfo, prs := d.m[funct]
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

	query = fmt.Sprintf(`from(bucket: "%s")
										|> range(start: -%dh)
  										|> filter(fn: (r) => r["_field"] == "duration" and r["completed"] == "true")
										|> group(columns: ["_measurement", "offloaded", "warm_start"])
										|> count()`, bucketName, 12)

	result, err = queryAPI.Query(context.Background(), query)
	if err == nil {
		// Iterate over query response
		for result.Next() {
			x := result.Record().Values()
			val := result.Record().Value().(int64)

			funct := x["_measurement"].(string)
			off := x["offloaded"].(string)
			warm_start := x["warm_start"].(string)

			location := LOCAL
			if off == "true" {
				location = OFFLOADED
			}

			fInfo, prs := d.m[funct]
			if !prs {
				continue
			}

			if warm_start == "true" {
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

	for _, fInfo := range d.m {
		for location := 0; location < 2; location++ {
			if fInfo.coldStartCount[location] == 0 {
				fInfo.probCold[location] = 1.0
			} else {
				fInfo.probCold[location] = float64(fInfo.coldStartCount[location]) / float64(fInfo.count[location]+fInfo.coldStartCount[location])
			}
		}
	}
}

func (d *decisionEngineFlux) handler() {
	evaluationTicker :=
		time.NewTicker(evaluationInterval)
	pcoldTicker :=
		time.NewTicker(time.Duration(config.GetInt(config.CONTAINER_EXPIRATION_TIME, 600)) * time.Second)

	for {
		select {
		case _ = <-evaluationTicker.C:
			s := rand.NewSource(time.Now().UnixNano())
			rGen = rand.New(s)
			log.Println("Evaluating")

			//Check if there are some instances with 0 arrivals
			for fName, fInfo := range d.m {
				for cName, cFInfo := range fInfo.invokingClasses {
					//Cleanup
					if cFInfo.arrivalCount == 0 {
						cFInfo.timeSlotsWithoutArrivals++
						if cFInfo.timeSlotsWithoutArrivals >= maxTimeSlots {
							d.Delete(fName, cName)
						}
					}
				}
			}

			d.queryDb()
			d.updateProbabilities()

		case r := <-requestChannel:
			var fKeys map[string]interface{}
			offloaded := "false"
			warmStart := "false"
			completed := "false"

			if !r.dropped {
				fKeys = map[string]interface{}{"duration": r.ExecReport.Duration, "init_time": r.ExecReport.InitTime}
				completed = "true"
			}

			if r.ExecReport.OffloadLatency != 0 {
				offloaded = "true"
				fKeys["offload_latency"] = r.ExecReport.OffloadLatency
			}

			if r.ExecReport.IsWarmStart {
				warmStart = "true"
			}

			p := influxdb2.NewPoint(r.Fun.Name,
				map[string]string{"class": r.ClassService.Name, "offloaded": offloaded, "warm_start": warmStart, "completed": completed},
				fKeys,
				time.Now())

			writeAPI.WritePoint(p)
		case arr := <-arrivalChannel:
			name := arr.Fun.Name

			fInfo, prs := d.m[name]
			if !prs {
				fInfo = &functionInfo{
					name:            name,
					memory:          arr.Fun.MemoryMB,
					cpu:             arr.Fun.CPUDemand,
					probCold:        [2]float64{1, 1},
					invokingClasses: make(map[string]*classFunctionInfo)}

				d.m[name] = fInfo
			}

			cFInfo, prs := fInfo.invokingClasses[arr.class]
			if !prs {
				cFInfo = &classFunctionInfo{functionInfo: fInfo,
					probExecute:              startingExecuteProb,
					probOffload:              startingOffloadProb,
					probDrop:                 1 - (startingExecuteProb + startingOffloadProb),
					arrivals:                 0,
					arrivalCount:             0,
					timeSlotsWithoutArrivals: 0,
					className:                arr.class}

				fInfo.invokingClasses[arr.class] = cFInfo
			}

			cFInfo.timeSlotsWithoutArrivals = 0
		case _ = <-pcoldTicker.C:
			//Reset arrivals for the time slot
			for _, fInfo := range d.m {
				fInfo.coldStartCount = [2]int64{0, 0}
				fInfo.timeSlotCount = [2]int64{0, 0}
			}
		}
	}
}

func (d *decisionEngineFlux) updateProbabilities() {
	solve(d.m)
}

func (d *decisionEngineFlux) ShowData() {
	for {
		time.Sleep(time.Second * 5)
		for _, fInfo := range d.m {
			for _, cFInfo := range fInfo.invokingClasses {
				log.Println(cFInfo)
			}
		}
	}
}

func (d *decisionEngineFlux) Completed(r *scheduledRequest, offloaded int) {
	requestChannel <- completedRequest{
		scheduledRequest: r,
		location:         offloaded,
		dropped:          false,
	}
}

func (d *decisionEngineFlux) Delete(function string, class string) {
	fInfo, prs := d.m[function]
	if !prs {
		return
	}

	delete(fInfo.invokingClasses, class)

	//If there aren't any more classes calls the function can be deleted
	if len(fInfo.invokingClasses) == 0 {
		delete(d.m, function)
	}
}

func (d *decisionEngineFlux) updateData(r completedRequest) {
	name := r.Fun.Name

	location := r.location

	fInfo, prs := d.m[name]
	//TODO maybe create here the entry in the function? Is it necessary?
	if !prs {
		return
	}

	if !r.ExecReport.IsWarmStart {
		diff := r.ExecReport.InitTime - fInfo.initTime[location]
		fInfo.initTime[location] = fInfo.initTime[location] +
			(1/float64(fInfo.count[location]))*(diff)

		fInfo.coldStartCount[location]++
	}
}
