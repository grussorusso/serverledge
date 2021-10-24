package main

import (
	"github.com/grussorusso/serverledge/internal/api"
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
	e.HideBanner = true
	e.Logger.Fatal(e.Start(":1323"))
}

func main() {
	containers.InitDockerContainerFactory()
	startAPIServer()
}
