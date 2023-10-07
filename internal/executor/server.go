package executor

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const resultFile = "/tmp/_executor_result.json"
const paramsFile = "/tmp/_executor.params"

func readExecutionResult(resultFile string) string {
	content, err := os.ReadFile(resultFile)
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
		os.Setenv("PARAMS_FILE", "")
	} else {
		paramsB, _ := json.Marshal(req.Params)
		err := os.WriteFile(paramsFile, paramsB, 0644)
		if err != nil {
			log.Printf("Could not write parameters to %s", paramsFile)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		os.Setenv("PARAMS_FILE", paramsFile)
	}

	// Exec handler process
	cmd := req.Command
	if cmd == nil || len(cmd) < 1 {
		// this request is either invalid or uses a custom runtime
		// in the latter case, we find the command in the env
		customCmd, ok := os.LookupEnv("CUSTOM_CMD")
		if !ok {
			log.Printf("Invalid request!\n")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cmd = strings.Split(customCmd, " ")
	}

	var resp *InvocationResult
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	out, err := execCmd.CombinedOutput()
	if err != nil {
		log.Printf("cmd.Run() failed with %s\n", err)
		fmt.Printf("Function output:\n%s\n", string(out)) // TODO: do something with output
		resp = &InvocationResult{Success: false}
	} else {
		result := readExecutionResult(resultFile)

		resp = &InvocationResult{true, result}
		fmt.Printf("Function output:\n%s\n", string(out)) // TODO: do something with output
	}

	w.Header().Set("Content-Type", "application/json")
	respBody, _ := json.Marshal(resp)
	w.Write(respBody)
}
