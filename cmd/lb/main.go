package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"golang.org/x/net/context"

	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/registration"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func startReverseProxy(e *echo.Echo) {
	e.Use(middleware.Recover())

	// Routes
	e.POST("/invoke/:fun", api.InvokeFunction)
	e.POST("/create", api.CreateFunction)
	e.POST("/delete", api.DeleteFunction)
	e.GET("/function", api.GetFunctions)
	e.GET("/status", api.GetServerStatus)

	// Start server
	portNumber := config.GetInt(config.API_PORT, 1323)
	e.HideBanner = true

	if err := e.Start(fmt.Sprintf(":%d", portNumber)); err != nil && err != http.ErrServerClosed {
		e.Logger.Fatal("shutting down the server")
	}
}

func registerTerminationHandler(e *echo.Echo) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	go func() {
		select {
		case sig := <-c:
			fmt.Printf("Got %s signal. Terminating...\n", sig)

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

	// register to etcd, this way server is visible to the others under a given local area
	registry := new(registration.Registry)
	fmt.Printf("%v", registry)

	e := echo.New()

	// Register a signal handler to cleanup things on termination
	registerTerminationHandler(e)

	startReverseProxy(e)

}
