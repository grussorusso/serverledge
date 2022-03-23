package executor

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"io/ioutil"
)

const resultFile = "/tmp/result.json"

func readExecutionResult(resultFile string) string {
	content, err := ioutil.ReadFile(resultFile)
	if err != nil {
		log.Printf("%v", err)
		return ""
	}

	return string(content)
}

func InvokeHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request
	reqDecoder := json.NewDecoder(r.Body)
	req := &InvocationRequest{}
	err := reqDecoder.Decode(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set environment variables
	os.Setenv("RESULT_FILE", resultFile)
	os.Setenv("HANDLER", req.Handler)
	os.Setenv("HANDLER_DIR", req.HandlerDir)
	params := req.Params
	if params == nil {
		os.Setenv("PARAMS", "{}")
	} else {
		paramsB, _ := json.Marshal(req.Params)
		os.Setenv("PARAMS", string(paramsB))
	}

	// Exec handler process
	cmd := req.Command
	if cmd == nil || len(cmd) < 1 {
		// this request is either invalid or uses a custom runtime
		// in the latter case, we find the command in the env
		customCmd, ok := os.LookupEnv("CUSTOM_CMD")
		if !ok {
			log.Printf("Invalid request!")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cmd = strings.Split(customCmd, " ")
	}

	t0 := time.Now()
	usage0 := new(syscall.Rusage)
	syscall.Getrusage(syscall.RUSAGE_CHILDREN, usage0)

	var resp *InvocationResult
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	out, err := execCmd.CombinedOutput()
	if err != nil {
		log.Printf("cmd.Run() failed with %s\n", err)
		fmt.Printf("Function output:\n%s\n", string(out)) // TODO: do something with output
		resp = &InvocationResult{Success: false}
	} else {
		result := readExecutionResult(resultFile)
		duration := time.Now().Sub(t0).Seconds()

		// get additional stats
		usage := new(syscall.Rusage)
		syscall.Getrusage(syscall.RUSAGE_CHILDREN, usage)
		cpuTimeNs := usage.Utime.Nano() + usage.Stime.Nano() - usage0.Utime.Nano() - usage0.Stime.Nano()
		cpuTime := float64(cpuTimeNs) / 1000000000.0

		resp = &InvocationResult{true, result, duration, cpuTime}
		fmt.Printf("Function output:\n%s\n", string(out)) // TODO: do something with output
	}

	w.Header().Set("Content-Type", "application/json")
	respBody, _ := json.Marshal(resp)
	w.Write(respBody)
}
