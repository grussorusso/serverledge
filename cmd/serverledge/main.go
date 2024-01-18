package main

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/grussorusso/serverledge/utils"
	"log"
	"os"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/metrics"
	"github.com/grussorusso/serverledge/internal/registration"
	"github.com/grussorusso/serverledge/internal/scheduling"
	"github.com/labstack/echo/v4"
)

func main() {
	configFileName := ""
	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}
	config.ReadConfiguration(configFileName)

	//setting up cache parameters
	api.CacheSetup()

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
		os.Exit(1)
	}

	// TODO: qui potrebbe servire un config.API_IP
	ip := config.GetString(config.API_IP, utils.GetIpAddress().String())
	url := fmt.Sprintf("http://%s:%d", ip, config.GetInt(config.API_PORT, 1323))
	myKey, err := registry.RegisterToEtcd(url)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	node.NodeIdentifier = myKey

	go metrics.Init()

	e := echo.New()

	// Register a signal handler to cleanup things on termination
	api.RegisterTerminationHandler(registry, e)

	schedulingPolicy := api.CreateSchedulingPolicy()
	go scheduling.Run(schedulingPolicy)

	if !isInCloud {
		err = registration.InitEdgeMonitoring(registry)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	}

	api.StartAPIServer(e)

}
