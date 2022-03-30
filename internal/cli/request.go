package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/client"

	"github.com/grussorusso/serverledge/utils"
)

type ParamsFlags map[string]string

func (i *ParamsFlags) String() string {
	return fmt.Sprintf("%q", *i)
}

func (i *ParamsFlags) Set(value string) error {
	tokens := strings.Split(value, ":")
	if len(tokens) < 2 {
		return fmt.Errorf("Invalid argument")
	}
	(*i)[tokens[0]] = strings.Join(tokens[1:], ":")
	return nil
}

func Invoke(funcName string, qosClass string, qosMaxRespT float64, params ParamsFlags) {
	if len(funcName) < 1 {
		fmt.Printf("Invalid function name.\n")
		ExitWithUsage()
	}

	// Prepare request
	request := client.InvocationRequest{
		Params:          params,
		QoSClass:        int64(api.DecodeServiceClass(qosClass)),
		QoSMaxRespT:     qosMaxRespT,
		CanDoOffloading: true}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		ExitWithUsage()
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
