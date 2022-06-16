package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/grussorusso/serverledge/internal/client"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/grussorusso/serverledge/internal/registration"
	"github.com/grussorusso/serverledge/utils"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/grussorusso/serverledge/internal/scheduling"
	"github.com/labstack/echo/v4"
)

var requestsPool = sync.Pool{
	New: func() any {
		return new(function.Request)
	},
}

// GetFunctions handles a request to list the function available in the system.
func GetFunctions(c echo.Context) error {
	list, err := function.GetAll()
	if err != nil {
		return c.String(http.StatusServiceUnavailable, "")
	}
	return c.JSON(http.StatusOK, list)
}

// InvokeFunction handles a function invocation request.
func InvokeFunction(c echo.Context) error {
	funcName := c.Param("fun")
	fun, ok := function.GetFunction(funcName)
	if !ok {
		log.Printf("Dropping request for unknown fun '%s'", funcName)
		return c.JSON(http.StatusNotFound, "")
	}

	var invocationRequest client.InvocationRequest
	err := json.NewDecoder(c.Request().Body).Decode(&invocationRequest)
	if err != nil && err != io.EOF {
		log.Printf("Could not parse request: %v", err)
		return fmt.Errorf("could not parse request: %v", err)
	}

	r := requestsPool.Get().(*function.Request)
	defer requestsPool.Put(r)
	r.Fun = fun
	r.Params = invocationRequest.Params
	r.Arrival = time.Now()
	r.Class = function.ServiceClass(invocationRequest.QoSClass)
	r.MaxRespT = invocationRequest.QoSMaxRespT
	r.CanDoOffloading = invocationRequest.CanDoOffloading
	// init fields if possibly not overwritten later
	r.ExecReport.SchedAction = ""
	r.ExecReport.OffloadLatency = 0.0

	err = scheduling.SubmitRequest(r)

	if errors.Is(err, node.OutOfResourcesErr) {
		return c.String(http.StatusTooManyRequests, "")
	} else if err != nil {
		log.Printf("Invocation failed: %v", err)
		return c.String(http.StatusInternalServerError, "")
	} else {
		return c.JSON(http.StatusOK, function.Response{ExecutionReport: r.ExecReport})
	}
}

// CreateFunction handles a function creation request.
func CreateFunction(c echo.Context) error {
	var f function.Function
	err := json.NewDecoder(c.Request().Body).Decode(&f)
	if err != nil && err != io.EOF {
		log.Printf("Could not parse request: %v", err)
		return err
	}

	_, ok := function.GetFunction(f.Name) // TODO: we would need a system-wide lock here...
	if ok {
		log.Printf("Dropping request for already existing function '%s'", f.Name)
		return c.JSON(http.StatusConflict, "")
	}

	log.Printf("New request: creation of %s", f.Name)

	// Check that the selected runtime exists
	if f.Runtime != container.CUSTOM_RUNTIME {
		_, ok := container.RuntimeToInfo[f.Runtime]
		if !ok {
			return c.JSON(http.StatusNotFound, "Invalid runtime.")
		}
	}

	err = f.SaveToEtcd()
	if err != nil {
		log.Printf("Failed creation: %v", err)
		return c.JSON(http.StatusServiceUnavailable, "")
	}
	response := struct{ Created string }{f.Name}
	return c.JSON(http.StatusOK, response)
}

// DeleteFunction handles a function deletion request.
func DeleteFunction(c echo.Context) error {
	var f function.Function
	err := json.NewDecoder(c.Request().Body).Decode(&f)
	if err != nil && err != io.EOF {
		log.Printf("Could not parse request: %v", err)
		return err
	}

	_, ok := function.GetFunction(f.Name) // TODO: we would need a system-wide lock here...
	if !ok {
		log.Printf("Dropping request for non existing function '%s'", f.Name)
		return c.JSON(http.StatusNotFound, "")
	}

	log.Printf("New request: deleting %s", f.Name)
	err = f.Delete()
	if err != nil {
		log.Printf("Failed deletion: %v", err)
		return c.JSON(http.StatusServiceUnavailable, "")
	}

	// Delete local warm containers
	node.ShutdownWarmContainersFor(&f)

	response := struct{ Deleted string }{f.Name}
	return c.JSON(http.StatusOK, response)
}

func DecodeServiceClass(serviceClass string) (p function.ServiceClass) {
	if serviceClass == "low" {
		return function.LOW
	} else if serviceClass == "performance" {
		return function.HIGH_PERFORMANCE
	} else if serviceClass == "availability" {
		return function.HIGH_AVAILABILITY
	} else {
		return function.LOW
	}
}

// GetServerStatus simple api to check the current server status
func GetServerStatus(c echo.Context) error {
	node.Resources.RLock()
	defer node.Resources.RUnlock()
	portNumber := config.GetInt("api.port", 1323)
	url := fmt.Sprintf("http://%s:%d", utils.GetIpAddress().String(), portNumber)
	response := registration.StatusInformation{
		Url:            url,
		AvailableMemMB: node.Resources.AvailableMemMB,
		AvailableCPUs:  node.Resources.AvailableCPUs,
		DropCount:      node.Resources.DropCount,
		Coordinates:    *registration.Reg.Client.GetCoordinate(),
	}

	return c.JSON(http.StatusOK, response)
}
