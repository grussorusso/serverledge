package main

import (
	"fmt"

	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/containers"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

func main() {
	config.ReadConfiguration()

	containers.InitDockerContainerFactory()

	startAPIServer()
}
