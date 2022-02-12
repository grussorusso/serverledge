package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/cache"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/scheduling"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func startAPIServer() {
	e := echo.New()
	e.Use(middleware.Recover())

	// Routes
	e.POST("/invoke/:fun", api.InvokeFunction)
	e.POST("/create", api.CreateFunction)
	e.POST("/delete", api.DeleteFunction)
	e.GET("/function", api.GetFunctions)

	// Start server
	portNumber := config.GetInt("api.port", 1323)
	e.HideBanner = true
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", portNumber)))
}

func cacheSetup() {
	//todo fix default values

	// setup cache space
	cache.Size = config.GetInt("cache.size", 10)

	//setup cleanup interval
	d := config.GetInt("cache.cleanup", 60)
	interval := time.Duration(d)
	cache.CleanupInterval = interval * time.Second

	//setup default expiration time
	d = config.GetInt("cache.expiration", 60)
	expirationInterval := time.Duration(d)
	cache.DefaultExp = expirationInterval * time.Second

	//cache first creation
	cache.GetCacheInstance()
}

func registerTerminationHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	go func() {
		select {
		case sig := <-c:
			fmt.Printf("Got %s signal. Terminating...\n", sig)
			scheduling.ShutdownAll()
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

	// Register a signal handler to cleanup things on termination
	registerTerminationHandler()

	go scheduling.Run()

	startAPIServer()
}
