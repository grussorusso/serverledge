package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/client"
	"github.com/spf13/cobra"

	"github.com/grussorusso/serverledge/utils"
)

func invoke(cmd *cobra.Command, args []string) {
	if len(funcName) < 1 {
		fmt.Printf("Invalid function name.\n")
		cmd.Help()
		return
	}

	paramsMap := make(map[string]string)
	for _, rawParam := range params {
		tokens := strings.Split(rawParam, ":")
		if len(tokens) < 2 {
			cmd.Help()
			return
		}
		paramsMap[tokens[0]] = strings.Join(tokens[1:], ":")
	}

	// Prepare request
	request := client.InvocationRequest{
		Params:          paramsMap,
		QoSClass:        int64(api.DecodeServiceClass(qosClass)),
		QoSMaxRespT:     qosMaxRespT,
		CanDoOffloading: true}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		cmd.Help()
		return
	}

	// Send invocation request
	url := fmt.Sprintf("http://%s:%d/invoke/%s", ServerConfig.Host, ServerConfig.Port, funcName)
	resp, err := utils.PostJson(url, invocationBody)
	if err != nil {
		fmt.Printf("Invocation failed: %v", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}
