package scheduling

import (
	"context"
	"fmt"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"log"
	"math/rand"
	"os"
	"time"
)

type decisionEnginePrometheus struct {
	m map[string]*functionInfoPrometheus
}

type functionInfoPrometheus struct {
	name string
	//Number of function requests
	count [2]int
	//Mean duration time
	meanDuration [2]float64
	//Variance of the duration time
	varianceDuration [2]float64
	//Count the number of cold starts to estimate probCold
	coldStartCount [2]int64
	//Number of requests that missed the deadline
	missed int
	//Average of init times when cold start
	initTime [2]float64
	//Memory requested by the function
	memory int64
	//CPU requested by the function
	cpu float64
	//Probability of a cold start when requesting the function
	probCold float64
	//TODO maybe put an array
	probColdOffload float64
	//Map containing information about function requests for every class
	invokingClasses map[string]*classFunctionInfoPrometheus
}

type classFunctionInfoPrometheus struct {
	//Pointer used for accessing information about the function
	*functionInfoPrometheus
	//
	probExecute float64
	probOffload float64
	probDrop    float64
	//
	share float64
}

func (dP *decisionEnginePrometheus) Decide(r *scheduledRequest) int {
	name := r.Fun.Name
	class := r.ClassService

	prob := rGen.Float64()

	var pe float64
	var po float64
	var pd float64

	var cFInfo *classFunctionInfoPrometheus

	arrivalChannel <- arrivalRequest{r, class.Name}

	fInfo, prs := dP.m[name]
	if !prs {
		pe = startingLocalProb
		po = startingCloudOffloadProb
		pd = 1 - (pe + po)
	} else {
		cFInfo, prs = fInfo.invokingClasses[class.Name]
		if !prs {
			pe = startingLocalProb
			po = startingCloudOffloadProb
			pd = 1 - (pe + po)
		} else {
			pe = cFInfo.probExecute
			po = cFInfo.probOffload
			pd = cFInfo.probDrop
		}
	}

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
		return LOCAL_EXEC_REQUEST
	} else if prob <= pe+po {
		log.Println("Execute OFFLOAD")
		return CLOUD_OFFLOAD_REQUEST
	} else {
		log.Println("Execute DROP")
		return DROP_REQUEST
	}
}

func (dP *decisionEnginePrometheus) InitDecisionEngine() {
	s := rand.NewSource(time.Now().UnixNano())
	rGen = rand.New(s)

	dP.m = make(map[string]*functionInfoPrometheus)

	go dP.ShowData()
	go dP.handler()
}

func getDataFromPrometheus() {
	identifier := node.NodeIdentifier
	log.Println(identifier)
	query := fmt.Sprintf("rate(sedge_exectime_sum{node='%s'}[1d])/rate(sedge_exectime_count{node='%s'}[1d])", identifier, identifier)
	log.Println(query)

	client, err := api.NewClient(api.Config{
		Address: "http://localhost:9090",
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}
	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//TODO edit range
	r := v1.Range{
		Start: time.Now().Add(-2 * time.Minute),
		End:   time.Now(),
		Step:  time.Second,
	}

	result, warnings, err := v1api.QueryRange(ctx, query, r, v1.WithTimeout(5*time.Second))
	if err != nil {
		fmt.Printf("Error querying Prometheus: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}

	log.Println(result.Type())
	log.Printf("Result:\n%v\n", result)

	mapData := make(map[string][]model.SampleValue)

	log.Println(result.(model.Matrix).Len())

	for i := 0; i < result.(model.Matrix).Len(); i++ {
		x := make([]model.SampleValue, 0)
		for _, val := range result.(model.Matrix)[0].Values {
			x = append(x, val.Value)
		}

		mapData[string(i)] = x
	}

	log.Println(mapData)
}

// TODO SYNC
func (dP *decisionEnginePrometheus) handler() {
	evaluationTicker :=
		time.NewTicker(time.Duration(config.GetInt(config.SOLVER_EVALUATION_INTERVAL, 10)) * time.Second)
	for {
		select {
		case _ = <-evaluationTicker.C:
			s := rand.NewSource(time.Now().UnixNano())
			rGen = rand.New(s)
			//log.Println("Evaluating")
			getDataFromPrometheus()
			dP.updateProbabilities()
		case arr := <-arrivalChannel:
			name := arr.Fun.Name

			fInfo, prs := dP.m[name]
			if !prs {
				fInfo = &functionInfoPrometheus{
					name:            name,
					memory:          arr.Fun.MemoryMB,
					cpu:             arr.Fun.CPUDemand,
					probCold:        1,
					invokingClasses: make(map[string]*classFunctionInfoPrometheus)}

				dP.m[name] = fInfo
			}

			cFInfo, prs := fInfo.invokingClasses[arr.class]
			if !prs {
				cFInfo = &classFunctionInfoPrometheus{functionInfoPrometheus: fInfo,
					probExecute: startingLocalProb,
					probOffload: startingCloudOffloadProb,
					probDrop:    1 - (startingLocalProb + startingCloudOffloadProb)}

				fInfo.invokingClasses[arr.class] = cFInfo
			}

			// TODO: aggiungi a fInfo il valore della dimensione del pacchetto (secondo me è diverso per ogni funzione)

		}
	}
}

func (dP *decisionEnginePrometheus) updateProbabilities() {
	//SolveProbabilities(dP.m)
}

func (dP *decisionEnginePrometheus) ShowData() {
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

func (dP *decisionEnginePrometheus) Delete(function string, class string) {
	fInfo, prs := dP.m[function]
	if !prs {
		return
	}

	delete(fInfo.invokingClasses, class)

	//If there aren't any more classes calls the function can be deleted
	if len(fInfo.invokingClasses) == 0 {
		delete(dP.m, function)
	}
}
