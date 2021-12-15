package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/grussorusso/serverledge/internal/config"

	"github.com/grussorusso/serverledge/internal/functions"
	"github.com/grussorusso/serverledge/internal/scheduling"
	"github.com/labstack/echo/v4"
)

// GetFunctions handles a request to list the functions available in the system.
func GetFunctions(c echo.Context) error {
	list, err := functions.GetAll()
	if err != nil {
		return c.String(http.StatusServiceUnavailable, "")
	}
	return c.JSON(http.StatusOK, list)
}

// InvokeFunction handles a function invocation request.
func InvokeFunction(c echo.Context) error {
	offloading := config.GetBool("offloading", false)
	funcName := c.Param("fun")
	function, ok := functions.GetFunction(funcName)
	if !ok {
		log.Printf("Dropping request for unknown function '%s'", funcName)
		return c.JSON(http.StatusNotFound, "")
	}

	var invocationRequest FunctionInvocationRequest
	err := json.NewDecoder(c.Request().Body).Decode(&invocationRequest)
	if err != nil && err != io.EOF {
		return fmt.Errorf("Could not parse request: %v", err)
	}

	r := &functions.Request{Fun: function, Params: invocationRequest.Params, Arrival: time.Now()}
	r.Class = invocationRequest.QoSClass
	r.MaxRespT = invocationRequest.QoSMaxRespT

	log.Printf("New request for function '%s' (class: %s, Max RespT: %f)", function, invocationRequest.QoSClass, invocationRequest.QoSMaxRespT)
	result, err := scheduling.Schedule(r)
	if err == nil {
		log.Printf("Request OK: %v", result.Result)
		return c.JSON(http.StatusOK, result.Result)
	} else if offloading {
		// offloading to handle missing resource status
		res, err := scheduling.Offload(r)
		defer res.Body.Close()
		if err == nil {
			body, _ := ioutil.ReadAll(res.Body)
			log.Printf("Offloading Request status OK: %s", string(body))
			return c.JSON(http.StatusOK, string(body))
		}
	}

	log.Printf("Failed invocation of %s: %v", function, err)
	return c.String(http.StatusServiceUnavailable, "")

}

// CreateFunction handles a function creation request.
func CreateFunction(c echo.Context) error {
	var f functions.Function
	err := json.NewDecoder(c.Request().Body).Decode(&f)
	if err != nil && err != io.EOF {
		log.Printf("Could not parse request: %v", err)
		return err
	}

	_, ok := functions.GetFunction(f.Name) // TODO: we would need a system-wide lock here...
	if ok {
		log.Printf("Dropping request for already existing function '%s'", f.Name)
		return c.JSON(http.StatusConflict, "")
	}

	log.Printf("New request: creation of %s", f.Name)
	err = f.SaveToEtcd()
	if err != nil {
		log.Printf("Failed creation: %v", err)
		return c.JSON(http.StatusServiceUnavailable, "")
	}
	response := struct{ Created string }{f.Name}
	return c.JSON(http.StatusOK, response)
}
