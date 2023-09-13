package test

import (
	"context"
	"fmt"
	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/metrics"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/grussorusso/serverledge/internal/registration"
	"github.com/grussorusso/serverledge/internal/scheduling"
	u "github.com/grussorusso/serverledge/utils"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc/codes"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"
)

const HOST = "127.0.0.1"
const PORT = 1323
const AREA = "ROME"

// use it to avoid running long running tests
const INTEGRATION_TEST = true

func testStartServerledge(isInCloud bool) (*registration.Registry, *echo.Echo) {

	//setting up cache parameters
	api.CacheSetup()
	schedulingPolicy := &scheduling.DefaultLocalPolicy{}
	// register to etcd, this way server is visible to the others under a given local area
	registry := new(registration.Registry)
	if isInCloud {
		registry.Area = "cloud/" + AREA
	} else {
		registry.Area = AREA
	}
	// before register checkout other servers into the local area
	//todo use this info later on; future work with active remote server selection
	_, err := registry.GetAll(true)
	if err != nil {
		log.Fatal(err)
	}

	ip := config.GetString(config.API_IP, u.GetIpAddress().String())
	url := fmt.Sprintf("http://%s:%d", ip, PORT)
	myKey, err := registry.RegisterToEtcd(url)
	if err != nil {
		log.Fatal(err)
	}

	node.NodeIdentifier = myKey

	go metrics.Init()

	e := echo.New()

	// Register a signal handler to cleanup things on termination
	api.RegisterTerminationHandler(registry, e)

	go scheduling.Run(schedulingPolicy)

	if !isInCloud {
		err = registration.InitEdgeMonitoring(registry)
		if err != nil {
			log.Fatal(err)
		}
	}
	// needed: if you call a function composition, internally will invoke each function
	go api.StartAPIServer(e)
	return registry, e

}

// current dir is ./serverledge/internal/fc
func TestMain(m *testing.M) {
	// spin up container with serverledge infrastructure
	if INTEGRATION_TEST {

		registry, echoServer, ok := setupServerledge()
		if ok != nil {
			fmt.Printf("failed to initialize serverledgde: %v\n", ok)
			os.Exit(int(codes.Internal))
		}

		// run all test independently
		code := m.Run()
		// tear down containers in order
		err := teardownServerledge(registry, echoServer)
		if err != nil {
			fmt.Printf("failed to remove serverledgde: %v\n", err)
			os.Exit(int(codes.Internal))
		}
		os.Exit(code)
	} else {
		code := m.Run()
		os.Exit(code)
	}
}

// startReliably can start the containers, or restart them if needed
func startReliably(startScript string, stopScript string, msg string) error {
	cmd := exec.CommandContext(context.Background(), "/bin/sh", startScript)
	err := cmd.Run()
	if err != nil {
		antiCmd := exec.CommandContext(context.Background(), "/bin/sh", stopScript)
		err = antiCmd.Run()
		if err != nil {
			return fmt.Errorf("stopping of %s failed", msg)
		}
		cmd = exec.CommandContext(context.Background(), "/bin/sh", startScript)
		err = cmd.Run()
	}
	if err == nil {
		fmt.Printf("%s started\n", msg)
	}
	return err
}

// run the bash script to initialize serverledge
func setupServerledge() (*registration.Registry, *echo.Echo, error) {
	err1 := startReliably("../../scripts/start-etcd.sh", "../../scripts/stop-etcd.sh", "ETCD")
	// Optional:
	//err2 := startReliably("../../scripts/start-influxdb.sh", "../../scripts/stop-influxdb.sh", "Influx")
	// err3 := startReliably("../../scripts/start-solver.sh", "../../scripts/stop-solver.sh", "Solver")

	registry, echoServer := testStartServerledge(false)
	return registry, echoServer, u.ReturnNonNilErr(err1) //, err2, err3)
}

// run the bash script to stop serverledge
func teardownServerledge(registry *registration.Registry, e *echo.Echo) error {
	cmd1 := exec.CommandContext(context.Background(), "/bin/sh", "../../scripts/stop-etcd.sh")
	// Optional:
	//cmd2 := exec.CommandContext(context.Background(), "/bin/sh", "../../scripts/stop-influxdb.sh")
	//cmd3 := exec.CommandContext(context.Background(), "/bin/sh", "../../scripts/stop-solver.sh")

	node.Resources.Lock()
	nContainers := len(node.Resources.ContainerPools)
	fmt.Printf("Terminating all %d containers...\n", nContainers)
	node.Resources.Unlock()

	node.ShutdownAllContainers()

	//stop container janitor
	node.StopJanitor()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	errEcho := e.Shutdown(ctx)

	errRegistry := registry.Deregister()
	err1 := cmd1.Run()
	fmt.Println("ETCD stopped")
	//err3 := cmd2.Run()
	//fmt.Println("Influx stopped")
	//err4 := cmd3.Run()
	//fmt.Println("Solver stopped")
	return u.ReturnNonNilErr(errEcho, errRegistry, err1) // , err2, err3)
}

// TestComposeFC checks the CREATE, GET and DELETE functionality of the Function Composition
func TestComposeFC(t *testing.T) {

	if !INTEGRATION_TEST {
		t.FailNow()
	}

	// GET1 - initially we do not have any function composition
	funcs, err := fc.GetAllFC()
	fmt.Println(funcs)
	lenFuncs := len(funcs)
	u.AssertNil(t, err)
	u.AssertEquals(t, 0, lenFuncs)

	fcName := "test"
	// CREATE - we create a test function composition
	m := make(map[string]interface{})
	m["input"] = 0
	length := 3
	_, fArr, err := initializeSameFunctionSlice(length, "js")
	u.AssertNil(t, err)

	dag, err := fc.CreateSequenceDag(fArr)
	u.AssertNil(t, err)

	fcomp := fc.NewFC(fcName, *dag, fArr, true)
	err2 := fcomp.SaveToEtcd()

	u.AssertNil(t, err2)

	// The creation is successful: we have one more function composition?
	// GET2
	funcs2, err3 := fc.GetAllFC()
	fmt.Println(funcs2)
	u.AssertNil(t, err3)
	u.AssertEquals(t, len(funcs2), lenFuncs+1)

	// the function is exactly the one i created?
	fun, ok := fc.GetFC(fcName)
	u.AssertTrue(t, ok)
	u.AssertTrue(t, fcomp.Equals(fun))

	// DELETE
	err4 := fcomp.Delete()
	u.AssertNil(t, err4)

	// The deletion is successful?
	// GET3
	funcs3, err5 := fc.GetAllFC()
	fmt.Println(funcs3)
	u.AssertNil(t, err5)
	u.AssertEquals(t, len(funcs3), lenFuncs)
}

// TestInvokeFC executes a Sequential Dag of length N, where each node executes a simple increment function.
func TestInvokeFC(t *testing.T) {

	if !INTEGRATION_TEST {
		t.FailNow()
	}

	fcName := "test"
	// CREATE - we create a test function composition
	length := 5
	f, fArr, err := initializeSameFunctionSlice(length, "js")
	u.AssertNil(t, err)
	dag, errDag := fc.CreateSequenceDag(fArr)
	u.AssertNil(t, errDag)
	fcomp := fc.NewFC(fcName, *dag, fArr, true)
	err1 := fcomp.SaveToEtcd()
	u.AssertNil(t, err1)

	// INVOKE - we call the function composition
	params := make(map[string]interface{})
	params[f.Signature.GetInputs()[0].Name] = 0 // FIXME: for javascript, the executor expects a string. But when you use "0", it seems to return null
	resultMap, err2 := fcomp.Invoke(params)
	u.AssertNil(t, err2)

	// check result
	output := resultMap.Result[f.Signature.GetOutputs()[0].Name]
	// res, errConv := strconv.Atoi(output.(string))
	u.AssertEquals(t, output, length)
	// u.AssertNil(t, errConv)
	fmt.Printf("%+v\n", resultMap)

	// cleaning up function composition and function
	err3 := fcomp.Delete()
	u.AssertNil(t, err3)
}

// TestInvokeChoiceFC executes a Choice Dag with N alternatives, and it executes only the second one. The functions are all the same increment function
func TestInvokeChoiceFC(t *testing.T) {

	if !INTEGRATION_TEST {
		t.FailNow()
	}

	fcName := "test"
	// CREATE - we create a test function composition
	length := 1
	input := 0
	f, fArr, err := initializeSameFunctionSlice(length, "py")
	u.AssertNil(t, err)

	conds := make([]fc.Condition, 3)
	conds[0] = fc.NewConstCondition(false)
	conds[1] = fc.NewSmallerCondition(2, 1)
	conds[2] = fc.NewConstCondition(true)

	dag, errDag := fc.CreateChoiceDag(conds, func() (*fc.Dag, error) { return fc.CreateSequenceDag(fArr) })
	u.AssertNil(t, errDag)
	fcomp := fc.NewFC(fcName, *dag, fArr, true)
	err1 := fcomp.SaveToEtcd()
	u.AssertNil(t, err1)

	// INVOKE - we call the function composition
	params := make(map[string]interface{})
	params[f.Signature.GetInputs()[0].Name] = input
	resultMap, err2 := fcomp.Invoke(params)
	u.AssertNil(t, err2)
	// checking the result, should be input + 1
	output := resultMap.Result[f.Signature.GetOutputs()[0].Name]

	u.AssertEquals(t, input+1, output)
	fmt.Printf("%+v\n", resultMap)

	// cleaning up function composition and function
	err3 := fcomp.Delete()
	u.AssertNil(t, err3)
}

// TestInvokeFC_DifferentFunctions executes a Sequential Dag of length 2, with two different functions
func TestInvokeFC_DifferentFunctions(t *testing.T) {

	if !INTEGRATION_TEST {
		t.FailNow()
	}

	fcName := "test"
	// CREATE - we create a test function composition
	fDouble, errF1 := initializePyFunction("double", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	u.AssertNil(t, errF1)

	fInc, errF2 := initializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	u.AssertNil(t, errF2)

	dag, errDag := fc.NewDagBuilder().
		AddSimpleNode(fDouble).
		AddSimpleNode(fInc).
		AddSimpleNode(fDouble).
		AddSimpleNode(fInc).
		Build()

	u.AssertNil(t, errDag)

	fcomp := fc.NewFC(fcName, *dag, []*function.Function{fDouble, fInc}, true)
	err1 := fcomp.SaveToEtcd()
	u.AssertNil(t, err1)

	// INVOKE - we call the function composition
	params := make(map[string]interface{})
	params[fDouble.Signature.GetInputs()[0].Name] = 2
	resultMap, err2 := fcomp.Invoke(params)
	if err2 != nil {
		log.Printf("%v\n", err2)
		t.FailNow()
	}
	u.AssertNil(t, err2)

	// check result
	output := resultMap.Result[fInc.Signature.GetOutputs()[0].Name]
	if output != 11 {
		t.FailNow()
	}

	// res, errConv := strconv.Atoi(output.(string))
	u.AssertEquals(t, (2*2+1)*2+1, output)
	// u.AssertNil(t, errConv)
	fmt.Println(resultMap)

	// cleaning up function composition and function
	err3 := fcomp.Delete()
	u.AssertNil(t, err3)
}
