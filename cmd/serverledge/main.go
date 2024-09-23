package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/grussorusso/serverledge/internal/node"

	"golang.org/x/net/context"

	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/cache"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/metrics"
	"github.com/grussorusso/serverledge/internal/registration"
	"github.com/grussorusso/serverledge/internal/scheduling"
	"github.com/grussorusso/serverledge/utils"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func startAPIServer(e *echo.Echo) {
	e.Use(middleware.Recover())

	// Routes
	e.POST("/invoke/:fun", api.InvokeFunction)
	e.POST("/prewarm", api.PrewarmFunction)
	e.POST("/create", api.CreateFunction)
	e.POST("/delete", api.DeleteFunction)
	e.GET("/function", api.GetFunctions)
	e.GET("/poll/:reqId", api.PollAsyncResult)
	e.GET("/status", api.GetServerStatus)

	// Start server
	portNumber := config.GetInt(config.API_PORT, 1323)
	e.HideBanner = true

	if err := e.Start(fmt.Sprintf(":%d", portNumber)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		e.Logger.Fatal("shutting down the server")
	}
}

func cacheSetup() {
	//todo fix default values

	// setup cache space
	cache.Size = config.GetInt(config.CACHE_SIZE, 10)

	//setup cleanup interval
	d := config.GetInt(config.CACHE_CLEANUP, 60)
	interval := time.Duration(d)
	cache.CleanupInterval = interval * time.Second

	//setup default expiration time
	d = config.GetInt(config.CACHE_ITEM_EXPIRATION, 60)
	expirationInterval := time.Duration(d)
	cache.DefaultExp = expirationInterval * time.Second

	//cache first creation
	cache.GetCacheInstance()
}

func registerTerminationHandler(r *registration.Registry, e *echo.Echo) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	go func() {
		select {
		case sig := <-c:
			fmt.Printf("Got %s signal. Terminating...\n", sig)
			node.ShutdownAllContainers()

			// deregister from etcd; server should be unreachable
			err := r.Deregister()
			if err != nil {
				log.Fatal(err)
			}

			//stop container janitor
			node.StopJanitor()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := e.Shutdown(ctx); err != nil {
				e.Logger.Fatal(err)
			}

			os.Exit(0)
		}
	}()
}

func main() {
	configFileName := ""
	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}
	config.ReadConfiguration(configFileName)

	//setting up cache parameters
	cacheSetup()

	// register to etcd, this way server is visible to the others under a given local area
	registry := new(registration.Registry)
	isInCloud := config.GetBool(config.IS_IN_CLOUD, false)
	if isInCloud {
		registry.Area = "cloud/" + config.GetString(config.REGISTRY_AREA, "ROME")
	} else {
		registry.Area = config.GetString(config.REGISTRY_AREA, "ROME")
	}
	// before register checkout other servers into the local area
	//todo use this info later on; future work with active remote server selection
	_, err := registry.GetAll(true)
	if err != nil {
		log.Fatal(err)
	}

	url := fmt.Sprintf("http://%s:%d", utils.GetIpAddress().String(), config.GetInt(config.API_PORT, 1323))
	myKey, err := registry.RegisterToEtcd(url)
	if err != nil {
		log.Fatal(err)
	}
	node.NodeIdentifier = myKey

	go metrics.Init()

	e := echo.New()

	// Register a signal handler to cleanup things on termination
	registerTerminationHandler(registry, e)

	schedulingPolicy := createSchedulingPolicy()
	go scheduling.Run(schedulingPolicy)

	if !isInCloud {
		err = registration.InitEdgeMonitoring(registry)
		if err != nil {
			log.Fatal(err)
		}
	}

	startAPIServer(e)

}

func createSchedulingPolicy() scheduling.Policy {
	policyConf := config.GetString(config.SCHEDULING_POLICY, "default")
	log.Printf("Configured policy: %s\n", policyConf)
	if policyConf == "cloudonly" {
		return &scheduling.CloudOnlyPolicy{}
	} else if policyConf == "edgecloud" {
		return &scheduling.CloudEdgePolicy{}
	} else if policyConf == "edgeonly" {
		return &scheduling.EdgePolicy{}
	} else {
		return &scheduling.DefaultLocalPolicy{}
	}
}
