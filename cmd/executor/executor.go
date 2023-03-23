package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/grussorusso/serverledge/internal/executor"
)

func main() {
	// We shall use two handlers in order to listen on different ports
	primaryHandler := http.NewServeMux()
	secondaryHandler := http.NewServeMux()

	primaryHandler.HandleFunc("/invoke", executor.InvokeHandler)
	go log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", executor.DEFAULT_EXECUTOR_PORT), primaryHandler))

	secondaryHandler.HandleFunc("/getFallbackAddresses", executor.GetFallbackAddresses)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", executor.DEFAULT_EXECUTOR_PORT+1), secondaryHandler))
}
