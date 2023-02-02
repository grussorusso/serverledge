package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/grussorusso/serverledge/internal/client"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/grussorusso/serverledge/internal/registration"
	"github.com/grussorusso/serverledge/utils"

	"github.com/grussorusso/serverledge/internal/scheduling"
	"github.com/labstack/echo/v4"
)

var requestsPool = sync.Pool{
	New: func() any {
		return new(function.Request)
	},
}

// Maximum amount of seconds to wait for a sync request migrated result. -1 means no limit
var MAX_SYNC_WAIT = config.GetInt(config.MAX_SYNC_WAIT_TIME, -1)

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
	r.Async = invocationRequest.Async
	r.ReqId = fmt.Sprintf("%s-%s%d", fun, node.NodeIdentifier[len(node.NodeIdentifier)-5:], r.Arrival.Nanosecond())
	// init fields if possibly not overwritten later
	r.ExecReport.SchedAction = ""
	r.ExecReport.OffloadLatency = 0.0
	r.ExecReport.Migrated = false

	if r.Async {
		go scheduling.SubmitAsyncRequest(r)
		return c.JSON(http.StatusOK, function.AsyncResponse{ReqId: r.ReqId})
	}

	err = scheduling.SubmitRequest(r)

	if errors.Is(err, node.OutOfResourcesErr) {
		return c.String(http.StatusTooManyRequests, "")
	} else if err != nil {
		log.Printf("Invocation failed: %v", err)
		return c.String(http.StatusInternalServerError, "")
	} else {
		// At this point there was no error submitting the request, but it is still possible that the container
		// has been migrated in the middle of its execution
		if r.ExecReport.Migrated {

			// If the execution has been migrated to another host, then wait until the other node
			// contacts back (until a timeout expires) or posts the result on ETCD

			var timeout int

			if MAX_SYNC_WAIT == -1 {
				timeout = 0
			} else {
				timeout = MAX_SYNC_WAIT
			}

			select {
			case res := <-scheduling.ResultsChannel:
				return c.JSON(http.StatusOK, function.Response{Success: true, ExecutionReport: res})
			case <-time.After(time.Duration(timeout) * time.Second):
				fmt.Println("Synchronous timeout exceeded, going to poll the result from ETCD")
			}

			// If the other node fails to contact, poll the result from ETCD
			etcdClient, err := utils.GetEtcdClient()
			if err != nil {
				log.Println("Could not connect to Etcd")
				return c.JSON(http.StatusInternalServerError, "")
			}
			// Acquire the connection to ETCD
			ctx := context.Background()
			key := fmt.Sprintf("async/%s", r.ReqId)

			// Define the wait-on-result variable depending on the timeout value (if set or no)
			payload := []byte{}
			total_waiting := 0
			var waitForResult bool
			if MAX_SYNC_WAIT == -1 {
				waitForResult = true
			} else {
				waitForResult = total_waiting < MAX_SYNC_WAIT
			}

			// Poll for the result until it's ready or the timeout expires
			for waitForResult {
				res, err := etcdClient.Get(ctx, key)
				if err != nil {
					log.Println(err)
					return c.JSON(http.StatusInternalServerError, "")
				}

				if len(res.Kvs) == 1 {
					// The result is ready. Leave the loop
					payload = res.Kvs[0].Value
					break
				}
				time.Sleep(1 * time.Second)
				if MAX_SYNC_WAIT != -1 {
					// Increment this only if the maximum wait has been set.
					total_waiting++
				}
			}
			return c.JSONBlob(http.StatusOK, payload)
		} else {
			// If the container wasn't migrated, send the execution report normally
			return c.JSON(http.StatusOK, function.Response{Success: true, ExecutionReport: r.ExecReport})
		}
	}
}

// PollAsyncResult checks for the result of an asynchronous invocation.
func PollAsyncResult(c echo.Context) error {
	reqId := c.Param("reqId")
	if len(reqId) < 0 {
		return c.JSON(http.StatusNotFound, "")
	}

	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Println("Could not connect to Etcd")
		return c.JSON(http.StatusInternalServerError, "")
	}

	ctx := context.Background()

	key := fmt.Sprintf("async/%s", reqId)
	res, err := etcdClient.Get(ctx, key)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "")
	}

	if len(res.Kvs) == 1 {
		payload := res.Kvs[0].Value
		return c.JSONBlob(http.StatusOK, payload)
	} else {
		return c.JSON(http.StatusNotFound, "")
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

func ReceiveResultAfterMigration(c echo.Context) error {
	err := scheduling.ReceiveResultAfterMigration(c)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func ReceiveContainerTar(c echo.Context) error {
	err := scheduling.ReceiveContainerTar(c)
	if err != nil {
		return fmt.Errorf("An error occurred receiving a container tar: %v", err)
	}
	return nil
}

func MigrationResponseListener(c echo.Context) error {
	err := scheduling.ReceiveResultFromNode(c)
	if err != nil {
		return fmt.Errorf("An error occurred receiving a result from a remote node: %v", err)
	}
	return nil
}
