package main

import (
	"fmt"
	"log"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	//"syscall"
	"io/ioutil"

	"github.com/grussorusso/serverledge/pkg/executor"
)

const APP_DIR = "/app"
const RESULT_FILE = "/tmp/result.json"

func readExecutionResult (resultFile string) string {
	content, err := ioutil.ReadFile(resultFile)
	if err != nil {
		log.Printf("%v", err)
		return ""
	}

	return string(content)
}

func invokeHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request
	d := json.NewDecoder(r.Body)
	req := &executor.InvocationRequest{}
	err := d.Decode(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set environment variables
	os.Setenv("RESULT_FILE", RESULT_FILE)
	os.Setenv("HANDLER", req.Handler)
	os.Setenv("HANDLER_DIR", req.HandlerDir)
	params := req.Params
	if params == nil {
		os.Setenv("PARAMS", "{}")
	} else {
		paramsB, _ := json.Marshal(req.Params)
		os.Setenv("PARAMS", string(paramsB))
	}

	log.Printf("Received request: %v", req)

	// Exec handler process
	cmd := req.Command
	if cmd == nil || len(cmd) < 1 {
		log.Printf("Invalid request!")
		return
	}
	//executable, lookErr := exec.LookPath(cmd[0])
	//if lookErr != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//log.Printf("Looked up executable: %s", executable)

	var resp *executor.InvocationResult
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	out, err := execCmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
		resp = &executor.InvocationResult{false, ""}
	} else {
		log.Printf("Handler command complete")
		result := readExecutionResult(RESULT_FILE)
		resp = &executor.InvocationResult{true, result}
		fmt.Printf("combined out:\n%s\n", string(out))
	}

	//env := os.Environ()
	//execErr := syscall.Exec(executable, cmd, env)
	//log.Printf("Executed cmd")
	//if execErr != nil {
	//	log.Printf("Handler command failed")
	//	resp = &executor.InvocationResult{false, ""}
	//} else {
	//	log.Printf("Handler command complete")
	//	result := readExecutionResult(RESULT_FILE)
	//	resp = &executor.InvocationResult{true, result}
	//}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("Sending response: %v", resp)
	respBody,_ := json.Marshal(resp)
	w.Write(respBody)
	//json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/invoke", invokeHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", executor.DEFAULT_EXECUTOR_PORT), nil))
}
