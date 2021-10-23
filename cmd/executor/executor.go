package main

import (
	"fmt"
	"log"
	"encoding/json"
	"net/http"
	"os"

	"github.com/grussorusso/serverledge/pkg/executor"
)

const APP_DIR = "/app"
const RESULT_FILE = "/tmp/result.json"

func invokeHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request
	d := json.NewDecoder(r.Body)
	req := &executor.InvocationRequest{}
	err := d.Decode(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// Set environment variables
	os.Setenv("RESULTS_FILE", "/tmp/result.json")
	os.Setenv("HANDLER", req.Handler)
	os.Setenv("HANDLER_DIR", req.HandlerDir)
	paramsB, _ := json.Marshal(req.Params)
	os.Setenv("PARAMS", string(paramsB))

	// Exec handler process
	// req.Command

	// Read result and return it
	resp := executor.InvocationResult{}
	// TODO

	fmt.Fprintf(w, "%v", resp)
}

func main() {
	http.HandleFunc("/invoke", invokeHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", executor.DEFAULT_EXECUTOR_PORT), nil))
}
