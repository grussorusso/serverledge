package main

import (
	"log"
	"net/http"
	"time"

	"github.com/grussorusso/serverledge/pkg/functions"
	"github.com/grussorusso/serverledge/pkg/scheduling"
	"github.com/grussorusso/serverledge/pkg/containers"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func getFunctions(c echo.Context) error {
	return c.JSON(http.StatusOK, "No functions in the system.")
}

func invokeFunction(c echo.Context) error {
	funcName := c.Param("fun")
	function, ok := functions.GetFunction(funcName)
	if !ok {
		log.Printf("Request for unknown function '%s'", funcName)
		return c.JSON(http.StatusNotFound, "")
	}
	r := &functions.Request{function, time.Now()}

	log.Printf("New request: %v", r)
	if result, err := scheduling.Schedule(r); err == nil {
		return c.JSON(http.StatusOK, result)
	} else {
		return c.JSON(http.StatusServiceUnavailable, "")
	}
}

func startAPIServer() {
	e := echo.New()
	e.Use(middleware.Recover())

	// Routes
	e.POST("/invoke/:fun", invokeFunction)
	//e.GET("/users/:id", getUser)
	e.GET("/functions", getFunctions)

	// Start server
	e.HideBanner = true
	e.Logger.Fatal(e.Start(":1323"))
}

func main() {
	containers.InitDockerContainerFactory()
	startAPIServer()
}
