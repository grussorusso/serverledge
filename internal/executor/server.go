package executor

import (
	"encoding/json"
	"errors"
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
		log.Printf("%v\n", err)
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
	err = os.Setenv("RESULT_FILE", resultFile)
	err = errors.Join(err, os.Setenv("HANDLER", req.Handler))
	err = errors.Join(err, os.Setenv("HANDLER_DIR", req.HandlerDir))
	params := req.Params
	if params == nil {
		err = errors.Join(err, os.Setenv("PARAMS_FILE", ""))
	} else {
		paramsB, _ := json.Marshal(req.Params)
		fileError := os.WriteFile(paramsFile, paramsB, 0644)
		if fileError != nil {
			log.Printf("Could not write parameters to %s\n", paramsFile)
			http.Error(w, fileError.Error(), http.StatusInternalServerError)
			return
		}
		err = errors.Join(err, os.Setenv("PARAMS_FILE", paramsFile))
	}
	if err != nil {
		log.Printf("Error while setting environment variables: %s\n", err)
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
		if req.ReturnOutput {
			resp = &InvocationResult{Success: false, Output: string(out)}
		} else {
			resp = &InvocationResult{Success: false, Output: ""}
		}
	} else {
		result := readExecutionResult(resultFile)

		if req.ReturnOutput {
			resp = &InvocationResult{true, result, string(out)}
		} else {
			resp = &InvocationResult{true, result, ""}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	respBody, _ := json.Marshal(resp)
	_, err = w.Write(respBody)
	if err != nil {
		log.Printf("Error while writing response to HTTP %s\n", err)
		return
	}
}
