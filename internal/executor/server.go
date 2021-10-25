package executor

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"

	"io/ioutil"
)

const appDir = "/app"
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
		log.Printf("Invalid request!")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resp *InvocationResult
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	_, err = execCmd.CombinedOutput()
	if err != nil {
		log.Printf("cmd.Run() failed with %s\n", err)
		resp = &InvocationResult{false, ""}
	} else {
		result := readExecutionResult(resultFile)
		resp = &InvocationResult{true, result}
		//fmt.Printf("combined out:\n%s\n", string(out)) // TODO: use output
	}

	w.Header().Set("Content-Type", "application/json")
	respBody, _ := json.Marshal(resp)
	w.Write(respBody)
}
