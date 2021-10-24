package api

import (
	"log"
	"net/http"
	"time"

	"github.com/grussorusso/serverledge/pkg/functions"
	"github.com/grussorusso/serverledge/pkg/scheduling"
	"github.com/labstack/echo/v4"
)

func GetFunctions(c echo.Context) error {
	return c.JSON(http.StatusOK, "No functions in the system.")
}

func InvokeFunction(c echo.Context) error {
	funcName := c.Param("fun")
	function, ok := functions.GetFunction(funcName)
	if !ok {
		log.Printf("Request for unknown function '%s'", funcName)
		return c.JSON(http.StatusNotFound, "")
	}
	// TODO: params
	r := &functions.Request{Fun: function, Arrival: time.Now()}

	log.Printf("New request: %v", r)
	if result, err := scheduling.Schedule(r); err == nil {
		log.Printf("Request OK: %s", result)
		return c.String(http.StatusOK, result)
	} else {
		log.Printf("Failed invocation of %s: %v", function, err)
		return c.String(http.StatusServiceUnavailable, "")
	}
}
