package test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/metrics"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/grussorusso/serverledge/internal/registration"
	"github.com/grussorusso/serverledge/internal/scheduling"
	u "github.com/grussorusso/serverledge/utils"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc/codes"
)

const HOST = "127.0.0.1"
const PORT = 1323
const AREA = "ROME"

func getShell() string {
	if IsWindows() {
		return "powershell.exe"
	} else {
		return "/bin/sh"
	}
}

func getShellExt() string {
	if IsWindows() {
		return ".bat"
	} else {
		return ".sh"
	}
}

var IntegrationTest bool

func testStartServerledge(isInCloud bool, outboundIp string) (*registration.Registry, *echo.Echo) {
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

	ip := config.GetString(config.API_IP, outboundIp)
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
	_, Experiment = os.LookupEnv("EXPERIMENT")
	// Parsing the test flags. Needed to ensure that the -short flag is parsed, so testing.Short() returns a nonNil bool
	flag.Parse()
	outboundIp, err := u.GetOutboundIp()
	if err != nil || outboundIp == nil {
		log.Fatalf("test cannot be executed without internet connection")
	}
	IntegrationTest = !testing.Short()

	// spin up container with serverledge infrastructure
	if IntegrationTest {
		//registry, echoServer, ok := setupServerledge(outboundIp.String())
		_, _, ok := setupServerledge(outboundIp.String())
		if ok != nil {
			fmt.Printf("failed to initialize serverledgde: %v\n", ok)
			os.Exit(int(codes.Internal))
		}

		// run all test independently
		code := m.Run()
		// tear down containers in order
		/*err := teardownServerledge(registry, echoServer)
		if err != nil {
			fmt.Printf("failed to remove serverledgde: %v\n", err)
			os.Exit(int(codes.Internal))
		}*/
		os.Exit(code)
	} else {
		code := m.Run()
		os.Exit(code)
	}
}

// startReliably can start the containers, or restart them if needed
func startReliably(startScript string, stopScript string, msg string) error {
	cmd := exec.CommandContext(context.Background(), getShell(), startScript)
	err := cmd.Run()
	if err != nil {
		antiCmd := exec.CommandContext(context.Background(), getShell(), stopScript)
		err = antiCmd.Run()
		if err != nil {
			return fmt.Errorf("stopping of %s failed", msg)
		}
		cmd = exec.CommandContext(context.Background(), getShell(), startScript)
		err = cmd.Run()
	}
	if err == nil {
		fmt.Printf("%s started\n", msg)
	}
	return err
}

// run the bash script to initialize serverledge
func setupServerledge(outboundIp string) (*registration.Registry, *echo.Echo, error) {
	err1 := startReliably("../../scripts/start-etcd"+getShellExt(), "../../scripts/stop-etcd"+getShellExt(), "ETCD")
	registry, echoServer := testStartServerledge(false, outboundIp)
	return registry, echoServer, u.ReturnNonNilErr(err1)
}

// run the bash script to stop serverledge
func teardownServerledge(registry *registration.Registry, e *echo.Echo) error {
	cmd1 := exec.CommandContext(context.Background(), getShell(), "../../scripts/remove-etcd"+getShellExt())

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
	fmt.Println("ETCD removed")
	return u.ReturnNonNilErr(errEcho, errRegistry, err1)
}
