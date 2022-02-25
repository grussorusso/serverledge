package cli

import (
	"fmt"
	"github.com/grussorusso/serverledge/utils"
	"net/http"
	"os"
)

func GetStatus() {
	// Send invocation request
	url := fmt.Sprintf("http://%s:%d/status", ServerConfig.Host, ServerConfig.Port)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Invocation failed: %v", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}
