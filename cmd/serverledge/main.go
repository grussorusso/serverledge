package main

import (
	"fmt"
	"github.com/grussorusso/serverledge/cache"
	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/containers"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"time"
)

func startAPIServer() {
	e := echo.New()
	e.Use(middleware.Recover())

	// Routes
	e.POST("/invoke/:fun", api.InvokeFunction)
	//e.GET("/users/:id", getUser)
	e.GET("/functions", api.GetFunctions)

	// Start server
	portNumber := config.GetInt("api.port", 1323)
	e.HideBanner = true
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", portNumber)))
}

func cacheSetup() {
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

func main() {
	config.ReadConfiguration()

	//setting up cache parameters
	cacheSetup()

	containers.InitDockerContainerFactory()

	startAPIServer()
}
