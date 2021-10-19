package main

import (
	"log"
	"net/http"
	"time"

	"github.com/grussorusso/serverledge/pkg/faas"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func getFunctions(c echo.Context) error {
	return c.JSON(http.StatusOK, "No functions in the system.")
}

func invokeFunction(c echo.Context) error {
	funcName := c.Param("fun")
	log.Printf("New request for function `%s`", funcName)

	r := &faas.Request{funcName, time.Now()}
	if err := faas.Schedule(r); err == nil {
		return c.JSON(http.StatusOK, "OK")
	} else {
		return c.JSON(http.StatusOK, "Failed")
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
	startAPIServer()
}
