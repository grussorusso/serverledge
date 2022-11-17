package scheduling

import (
	"context"
	"fmt"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
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

	log.Println("Probabilities are", pe, po, pd)

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
		return DROP_REQUEST
	}
}

func (d *decisionEngineFlux) InitDecisionEngine() {
	s := rand.NewSource(time.Now().UnixNano())
	rGen = rand.New(s)

	// TODO edit batch size and token
	clientInflux = influxdb2.NewClientWithOptions("http://localhost:8086", "my-token",
		influxdb2.DefaultOptions().SetBatchSize(20))
	// TODO edit bucket with node info
	//								NODE			  INFO(arrivals, completions)
	// "serverledge-"+node.NodeIdentifier
	//TODO create custom bucket
	/*
		_, err := clientInflux.BucketsAPI().CreateBucket(context.Background(), &domain.Bucket{
			Name: "completions",
		})
		if err != nil {
				return
		}

	*/

	writeAPI = clientInflux.WriteAPI("serverledge", "completions")
	queryAPI = clientInflux.QueryAPI("serverledge")

	evaluationInterval = time.Duration(config.GetInt(config.SOLVER_EVALUATION_INTERVAL, 10)) * time.Second

	d.m = make(map[string]*functionInfo)

	go d.ShowData()
	go d.handler()
}

func (d *decisionEngineFlux) queryDb() {

	query := fmt.Sprintf(`from(bucket: "completions")
										|> range(start: -%dh)
										|> group(columns: ["_measurement", "offloaded"])
										|> filter(fn: (r) => r["_field"] == "duration")
										|> tail(n: %d)
										|> exponentialMovingAverage(n: %d)`, 12, 100, 100)

	result, err := queryAPI.Query(context.Background(), query)
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
		log.Println(err)
	}

	query = fmt.Sprintf(`from(bucket: "completions")
										|> range(start: -%dh)
										|> filter(fn: (r) => r["_field"] == "offload_latency")
										|> group()
										|> tail(n: %d)
										|> exponentialMovingAverage(n: %d)`, 12, 100, 100)

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
		log.Println(err)
	}

	query = fmt.Sprintf(`from(bucket: "completions")
										|> range(start: -%dh)
										|> group(columns: ["_measurement", "offloaded"])
										|> filter(fn: (r) => r["_field"] == "init_time" and r["warm_start"] == "false")
										|> tail(n: %d)
										|> exponentialMovingAverage(n: %d)`, 12, 100, 100)
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
		log.Println(err)
	}

	query = fmt.Sprintf(`from(bucket: "completions")
										|> range(start: -%dh)
										|> group(columns: ["_measurement", "offloaded", "warm_start"])
										|> count()`, 12)

	result, err = queryAPI.Query(context.Background(), query)
	if err == nil {
		// Iterate over query response
		for result.Next() {
			x := result.Record().Values()
			val := result.Record().Value().(int64)

			log.Println(x)
			log.Println(val)
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
		log.Println(err)
	}

	for _, fInfo := range d.m {
		for location := 0; location < 2; location++ {
			if fInfo.coldStartCount[location] == 0 {
				fInfo.probCold[location] = 1.0
			} else {
				fInfo.probCold[location] = float64(fInfo.coldStartCount[location]) / float64(fInfo.count[location])
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

			//Reset arrivals for the time slot
			for _, fInfo := range d.m {
				for _, cFInfo := range fInfo.invokingClasses {
					//Cleanup
					cFInfo.arrivalCount = 0
					cFInfo.arrivals = 0
				}
			}

		case r := <-requestChannel:
			offloaded := "false"
			warmStart := "false"

			fKeys := map[string]interface{}{"duration": r.ExecReport.Duration, "init_time": r.ExecReport.InitTime}

			if r.ExecReport.OffloadLatency != 0 {
				offloaded = "true"
				fKeys["offload_latency"] = r.ExecReport.OffloadLatency
			}

			if r.ExecReport.IsWarmStart {
				warmStart = "true"
			}

			p := influxdb2.NewPoint(r.Fun.Name,
				map[string]string{"class": r.ClassService.Name, "offloaded": offloaded, "warm_start": warmStart},
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

			cFInfo.arrivalCount++
			cFInfo.arrivals = cFInfo.arrivalCount / float64(evaluationInterval)
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

}

func (d *decisionEngineFlux) Completed(r *scheduledRequest, offloaded int) {
	requestChannel <- completedRequest{
		scheduledRequest: r,
		location:         offloaded,
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
