package main

import (
	"fmt"
	"log"
	"net/http"
)

const DEFAULT_EXECUTOR_PORT = 8080
const APP_DIR = "/app"
const RESULT_FILE = "/tmp/result.json"

func invokeHandler(w http.ResponseWriter, r *http.Request) {
	// Set environment variables

	// Exec handler process

	// Read result and return it
	// TODO

	fmt.Fprintf(w, "Result = ???")
}

func main() {
	http.HandleFunc("/invoke", invokeHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", DEFAULT_EXECUTOR_PORT), nil))
}
