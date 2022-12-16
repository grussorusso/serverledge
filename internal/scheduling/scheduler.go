package scheduling

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/grussorusso/serverledge/internal/metrics"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/labstack/echo/v4"

	"github.com/grussorusso/serverledge/internal/config"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
)

var requests chan *scheduledRequest
var completions chan *completion

var remoteServerUrl string
var executionLogEnabled bool

var offloadingClient *http.Client
var checkpointArchiveSizeLimit = 10 * 1024
var checkpointFormField = "checkpoint"

var restorePool = sync.Pool{
	New: func() any {
		return new(function.Request)
	},
}

func Run(p Policy) {
	requests = make(chan *scheduledRequest, 500)
	completions = make(chan *completion, 500)

	// initialize Resources resources
	availableCores := runtime.NumCPU()
	node.Resources.AvailableMemMB = int64(config.GetInt(config.POOL_MEMORY_MB, 1024))
	node.Resources.AvailableCPUs = config.GetFloat(config.POOL_CPUS, float64(availableCores))
	node.Resources.ContainerPools = make(map[string]*node.ContainerPool)
	log.Printf("Current resources: %v", node.Resources)

	if config.DEFAULT_CONTAINER_MANAGER == "docker" {
		container.InitDockerContainerFactory()
	} else if config.DEFAULT_CONTAINER_MANAGER == "podman" {
		container.InitPodmanContainerFactory()
	} else {
		log.Fatal("An invalid container manager was specified in the configuration file.")
		return
	}

	//janitor periodically remove expired warm container
	node.GetJanitorInstance()

	tr := &http.Transport{
		MaxIdleConns:        2500,
		MaxIdleConnsPerHost: 2500,
		MaxConnsPerHost:     0,
		IdleConnTimeout:     30 * time.Minute,
	}
	offloadingClient = &http.Client{Transport: tr}

	// initialize scheduling policy
	p.Init()

	remoteServerUrl = config.GetString(config.CLOUD_URL, "")

	log.Println("Scheduler started.")

	var r *scheduledRequest
	var c *completion
	for {
		select {
		case r = <-requests:
			go p.OnArrival(r)
		case c = <-completions:
			node.ReleaseContainer(c.contID, c.Fun)
			p.OnCompletion(r)

			if metrics.Enabled {
				metrics.AddCompletedInvocation(r.Fun.Name)
				if r.ExecReport.SchedAction != SCHED_ACTION_OFFLOAD {
					metrics.AddFunctionDurationValue(r.Fun.Name, r.ExecReport.Duration)
				}
			}
		}
	}

}

// SubmitRequest submits a newly arrived request for scheduling and execution
func SubmitRequest(r *function.Request) error {
	schedRequest := scheduledRequest{
		Request:         r,
		decisionChannel: make(chan schedDecision, 1)}
	requests <- &schedRequest

	// wait on channel for scheduling action
	schedDecision, ok := <-schedRequest.decisionChannel
	if !ok {
		return fmt.Errorf("could not schedule the request")
	}
	//log.Printf("[%s] Scheduling decision: %v", r, schedDecision)

	var err error
	if schedDecision.action == DROP {
		//log.Printf("[%s] Dropping request", r)
		return node.OutOfResourcesErr
	} else if schedDecision.action == EXEC_REMOTE {
		//log.Printf("Offloading request")
		err = Offload(r, schedDecision.remoteHost)
		if err != nil {
			return err
		}
	} else {
		err = Execute(schedDecision.contID, &schedRequest)
		if err != nil {
			return err
		}
	}
	return nil
}

// SubmitAsyncRequest submits a newly arrived async request for scheduling and execution
func SubmitAsyncRequest(r *function.Request) {
	schedRequest := scheduledRequest{
		Request:         r,
		decisionChannel: make(chan schedDecision, 1)}
	requests <- &schedRequest

	// wait on channel for scheduling action
	schedDecision, ok := <-schedRequest.decisionChannel
	if !ok {
		publishAsyncResponse(r.ReqId, function.Response{Success: false})
		return
	}

	var err error
	if schedDecision.action == DROP {
		publishAsyncResponse(r.ReqId, function.Response{Success: false})
	} else if schedDecision.action == EXEC_REMOTE {
		//log.Printf("Offloading request")
		err = OffloadAsync(r, schedDecision.remoteHost)
		if err != nil {
			publishAsyncResponse(r.ReqId, function.Response{Success: false})
		}
	} else {

		err = Execute(schedDecision.contID, &schedRequest)
		if err != nil {
			publishAsyncResponse(r.ReqId, function.Response{Success: false})
		}
		publishAsyncResponse(r.ReqId, function.Response{Success: true, ExecutionReport: r.ExecReport})

		/*----
		DEMO - Migration process: Let's suppose a migration decision is taken.
		----
		shouldMigrate := true
		fallbackAddresses := []string{"IP1", "IP2", "10.0.2.6"}
		// When the function execution is called, a migration occurs at the same time
		go Execute(schedDecision.contID, &schedRequest)
		if shouldMigrate {
			do_migration_demo(schedDecision.contID, fallbackAddresses)
		}*/
	}
}

func do_migration_demo(contID container.ContainerID, fallbackAddresses []string) error {
	checkpointArchiveName := contID + ".tar.gz"
	// First of all, checkpoint the container (specifying the fallback addresses)
	err := Checkpoint(contID, fallbackAddresses)
	if err != nil {
		return fmt.Errorf("An error occurred while trying to checkpoint the container: %v", err)
	}

	// Try to send the checkpoint .tar file to every candidate
	for _, migrationCandidateIP := range fallbackAddresses {
		url := fmt.Sprintf("http://%s:%d/receiveContainerTar", migrationCandidateIP, 1323)
		err = prepareAndSendContainerTar(url, checkpointArchiveName)
		if err != nil {
			fmt.Println("ERR: Could not send req to ", migrationCandidateIP, "\n-> ", err)
		} else {
			fmt.Println("\t...Checkpoint sent to ", migrationCandidateIP)
			break
		}
	}
	return err
}

func ReceiveResultAfterMigration(c echo.Context) error {
	b, _ := io.ReadAll(c.Request().Body)
	b = bytes.Trim(b, "\x00")
	result := strings.Trim(string(b), "\x00")
	//if err != nil {
	//	return fmt.Errorf("An error occurred receiving result after migration: %v", err)
	//}
	fmt.Println("Il risultato arrivato è\n", result)
	return nil
}

func ReceiveContainerTar(c echo.Context) error {
	r := c.Request()
	r.ParseMultipartForm(int64(checkpointArchiveSizeLimit))
	file, handler, err := r.FormFile(checkpointFormField) // Get the form file
	if err != nil {
		fmt.Println("An error occurred while trying to acquire the tar: ", err)
		return err
	}
	defer file.Close()

	fmt.Printf("Uploaded file specs:\nName -> %+v\nSize -> %+v\nMIME Header -> %+v\n", handler.Filename, handler.Size, handler.Header)
	currDir, _ := os.Getwd()
	tempFile, err := ioutil.TempFile(currDir, "checkpoint-*.tar.gz") // Prepare the temporary file
	if err != nil {
		fmt.Println("An error occurred preparing the temporary file: ", err)
		return err
	}
	defer tempFile.Close()

	fileBytes, _ := ioutil.ReadAll(file) // Read file content in a byte array
	tempFile.Write(fileBytes)            // Write the byte array in the temporary file
	fmt.Printf("Checkpoint file %s successfully received.\n", tempFile.Name())

	err = scheduleRestore(tempFile.Name())

	return err
}

func prepareAndSendContainerTar(url string, checkpointArchiveName string) error {
	fileDir, _ := os.Getwd() // Get current path
	filePath := path.Join(fileDir, checkpointArchiveName)

	file, _ := os.Open(filePath) // Open file
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile(checkpointFormField, filepath.Base(file.Name()))
	io.Copy(part, file) // Copy file bytes in a multipart form data file
	writer.Close()
	r, _ := http.NewRequest("POST", url, body)
	r.Header.Add("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	_, err := client.Do(r)
	return err
}

func scheduleRestore(archiveName string) error {
	restoreChannel := make(chan schedDecision, 1)
	// wait on channel for scheduling action
	schedDecision, ok := <-restoreChannel
	if !ok {
		return fmt.Errorf("could not schedule the restore operation")
	}
	if schedDecision.action == DROP {
		return node.OutOfResourcesErr
	}
	err := Restore("restored-"+archiveName, archiveName)
	if err != nil {
		return fmt.Errorf("An error occurred restoring the checkpoint tar: %v", err)
	}
	return err

}

func handleColdStart(r *scheduledRequest) (isSuccess bool) {
	newContainer, err := node.NewContainer(r.Fun)
	if errors.Is(err, node.OutOfResourcesErr) || err != nil {
		log.Printf("Cold start failed: %v", err)
		return false
	} else {
		execLocally(r, newContainer, false)
		return true
	}
}

func dropRequest(r *scheduledRequest) {
	r.decisionChannel <- schedDecision{action: DROP}
}

func execLocally(r *scheduledRequest, c container.ContainerID, warmStart bool) {
	initTime := time.Now().Sub(r.Arrival).Seconds()
	r.ExecReport.InitTime = initTime
	r.ExecReport.IsWarmStart = warmStart

	decision := schedDecision{action: EXEC_LOCAL, contID: c}
	r.decisionChannel <- decision
}

func handleOffload(r *scheduledRequest, serverHost string) {
	r.CanDoOffloading = false // the next server can't offload this request
	r.decisionChannel <- schedDecision{
		action:     EXEC_REMOTE,
		contID:     "",
		remoteHost: serverHost,
	}
}

func handleCloudOffload(r *scheduledRequest) {
	cloudAddress := config.GetString(config.CLOUD_URL, "")
	handleOffload(r, cloudAddress)
}
