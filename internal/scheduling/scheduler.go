package scheduling

import (
	"bytes"
	"encoding/json"
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

	"github.com/grussorusso/serverledge/internal/executor"
	"github.com/grussorusso/serverledge/internal/metrics"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/labstack/echo/v4"

	"github.com/grussorusso/serverledge/internal/config"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
)

var requests chan *scheduledRequest
var restores chan *scheduledRestore
var completions chan *completion

var ResultsChannel chan function.ExecutionReport
var migrationAddresses chan string

var remoteServerUrl string
var executionLogEnabled bool

var offloadingClient *http.Client
var checkpointArchiveSizeLimit = 10 * 1024
var checkpointFormField = "checkpoint"

var nodeRestoredContainers chan container.ContainerID

var restorePool = sync.Pool{
	New: func() any {
		return new(function.Request)
	},
}

func Run(p Policy) {
	//Let's initialize the channels for inter process communication
	requests = make(chan *scheduledRequest, 500)
	completions = make(chan *completion, 500)
	restores = make(chan *scheduledRestore, 500)
	// We'll need a channel to send the results after a migration as well
	ResultsChannel = make(chan function.ExecutionReport, 1)
	// And a channel to share the migration client ip among processes
	migrationAddresses = make(chan string, 1)
	// This map associates all node's requests to their container
	node.NodeRequests = make(map[string]executor.InvocationRequest)
	// Channel to retreive the restore container IDs
	nodeRestoredContainers = make(chan container.ContainerID, 500)

	// initialize Resources resources
	availableCores := runtime.NumCPU()
	node.Resources.AvailableMemMB = int64(config.GetInt(config.POOL_MEMORY_MB, 1024))
	node.Resources.AvailableCPUs = config.GetFloat(config.POOL_CPUS, float64(availableCores))
	node.Resources.ContainerPools = make(map[string]*node.ContainerPool)
	log.Printf("Current resources: %v", node.Resources)

	containerManager := config.GetString(config.DEFAULT_CONTAINER_MANAGER, "podman")
	if containerManager == "docker" {
		container.InitDockerContainerFactory()
	} else if containerManager == "podman" {
		container.InitPodmanContainerFactory()
	} else {
		log.Fatal("An invalid container manager was specified in the configuration file.")
		return
	}

	// Start the thread that monitors node's memory, in order to migrate something if necessary
	if config.GetBool(config.ALLOW_MIGRATION, true) {
		go startMigrationMonitor()
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
	var restore *scheduledRestore

	for {
		select {
		case r = <-requests:
			go p.OnArrival(r)
		case c = <-completions:
			node.ReleaseContainer(c.contID, c.Fun)
			p.OnCompletion(c.scheduledRequest)

			if metrics.Enabled {
				metrics.AddCompletedInvocation(c.Fun.Name)
				if c.ExecReport.SchedAction != SCHED_ACTION_OFFLOAD {
					metrics.AddFunctionDurationValue(c.Fun.Name, c.ExecReport.Duration)
				}
			}
		case restore = <-restores:
			p.OnRestore(restore)
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
		/*-------------------------------------------------------------------------------
		DEMO - Migration process: Let's suppose a migration decision is taken.
		When the function execution is called, a migration occurs at the same time
		----*/
		//go Execute(schedDecision.contID, &schedRequest)
		//migration_demo(r, schedDecision.contID)
		//-------------------------------------------------------------------------------*/
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
		/*-------------------------------------------------------------------------------
		DEMO - Migration process: Let's suppose a migration decision is taken.
		When the function execution is called, a migration occurs at the same time
		----*/
		//go Execute(schedDecision.contID, &schedRequest)
		//migration_demo(r, schedDecision.contID)
		//-------------------------------------------------------------------------------*/
	}
}

// Start a migration process
func Migrate(contID container.ContainerID, fallbackAddresses []string) error {
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

	node.Resources.Lock()
	node.Resources.AvailableMemMB += node.NodeRequests[contID].OriginalRequest.Fun.MemoryMB
	node.Resources.AvailableCPUs += node.NodeRequests[contID].OriginalRequest.Fun.CPUDemand
	node.Resources.Unlock()

	file, err := os.OpenFile("timelogA.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString(time.Now().String() + "\n")
	return err
}

// Listen on a port to receive the checkpointed container archive
func ReceiveContainerTar(c echo.Context) error {

	fil, err := os.OpenFile("timelogB.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	fil.WriteString(time.Now().String() + "\n")

	// First of all notify the presence of a client migrator ip
	migrationAddresses <- c.RealIP()
	// Then work to retrieve the checkpointed container archive
	r := c.Request()
	r.ParseMultipartForm(int64(checkpointArchiveSizeLimit))
	file, handler, err := r.FormFile(checkpointFormField) // Get the form file
	if err != nil {
		fmt.Println("An error occurred while trying to acquire the tar: ", err)
		return err
	}
	defer file.Close()

	fmt.Printf("File received. Specs:\nName -> %+v\nSize -> %+v\nMIME Header -> %+v\n", handler.Filename, handler.Size, handler.Header)
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

	contID, err := scheduleRestore(tempFile.Name())
	// Put the container on the restore channel
	nodeRestoredContainers <- contID
	return err
}

// Listen on a port to receive the result from a restored container
func ReceiveResultAfterMigration(c echo.Context) error {
	b, _ := io.ReadAll(c.Request().Body) // Get the result
	result := getMigrationResult(b)      // Create the struct from it
	if result.Error != nil {
		return fmt.Errorf("An error occurred during migration result unmarshaling: %v", result.Error)
	}
	report := &function.ExecutionReport{Result: result.Result, Migrated: true, Id: result.Id, Class: result.Class} // Build the report struct
	fmt.Printf("A result has been received from a migrated container: %s\nRequest ID: %s", report.Result, report.Id)

	//Before uploading the result to ETCD, try to contact back the node (synchronous case)
	originalNodeIP := retrieveOriginalNodeIP()
	if originalNodeIP != "" {
		//If the call was synchronous
		url := fmt.Sprintf("http://%s:%d/migrationResponseListener", originalNodeIP, 1323)
		postBody, _ := json.Marshal(report)
		postBodyB := bytes.NewBuffer(postBody)
		_, err := http.Post(url, "application/json", postBodyB)
		if err != nil {
			fmt.Printf("Error contacting primary node: %v\n", err)
			return err
		}
	}
	publishAsyncResponse(result.Id, function.Response{Success: true, ExecutionReport: *report}) // Send the result to etcd
	fmt.Println("Result stored on ETCD.")
	// Retrieve the container Id in order to remove it from the requests
	contID := <-nodeRestoredContainers
	container.Destroy(contID)
	return nil
}

// Listen on a port to receive the result from the node which restored the migrated container
func ReceiveResultFromNode(c echo.Context) error {
	b, _ := io.ReadAll(c.Request().Body) // Get the result
	var res function.ExecutionReport
	err := json.Unmarshal(b, &res)
	if err != nil {
		fmt.Println("Error unmarshalling result from the migrated node")
		return nil
	}
	node.Resources.Lock()
	// Retrieve the container Id from the request Id, in order to remove it from the requests
	var contID string
	for id, request := range node.NodeRequests {
		if request.Id == res.Id {
			contID = id
			break
		}
	}
	delete(node.NodeRequests, contID)
	node.Resources.Unlock()

	// Publish the result on the channel, for the API waiting to respond to the client
	ResultsChannel <- res
	return nil
}

// Build and send the request containing the container checkpoint archive, in order to send it to the remote node
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
	_, err := client.Do(r) // Send the request
	return err
}

// Schedule a restore operation
func scheduleRestore(archiveName string) (string, error) {
	// Create a restore request for a given container, from a given archive.
	restoreRequest := scheduledRestore{
		contID:         "restored-" + archiveName,
		archiveName:    archiveName,
		restoreChannel: make(chan restoreResult, 1)}
	// Add the request to the channel
	restores <- &restoreRequest

	// Wait on the channel for the restore to be executed
	restoreResponse := <-restoreRequest.restoreChannel
	if restoreResponse.err != nil {
		return "", fmt.Errorf("An error occurred restoring the checkpoint tar: %v", restoreResponse.err)
	}
	return restoreResponse.contID, nil
}

// Translate the received result from the restored container into the expected format
func getMigrationResult(b []byte) executor.MigrationResult {
	// Manipulate the result string to remove noise and null bytes
	result := strings.Trim(string(bytes.Trim(b, "\x00")), "\x00")
	result = strings.Replace(result, "\\\\\\\"", "", -1)
	result = strings.Replace(result, "\\", "", -1)
	result = result[1 : len(result)-1]

	// Build the json containing the informations about the function execution after migration
	var res executor.MigrationResult
	err := json.Unmarshal([]byte(result), &res)
	if err != nil {
		res.Error = err
		return res
	}
	fmt.Println("Received data:\nResult: ", res.Result, "\nId: ", res.Id, "\nSuccess: ", res.Success, "\nClass: ", res.Class)
	return res
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
