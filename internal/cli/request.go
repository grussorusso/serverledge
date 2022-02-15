package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/function"
	"os"
	"strings"

	"github.com/grussorusso/serverledge/utils"
)

type paramsFlags map[string]string

func (i *paramsFlags) String() string {
	return fmt.Sprintf("%q", *i)
}

func (i *paramsFlags) Set(value string) error {
	tokens := strings.Split(value, ":")
	if len(tokens) != 2 {
		return fmt.Errorf("Invalid argument")
	}
	(*i)[tokens[0]] = tokens[1]
	return nil
}

func Invoke() {
	var params paramsFlags = make(map[string]string)

	invokeCmd := flag.NewFlagSet("invoke", flag.ExitOnError)
	funcName := invokeCmd.String("function", "", "name of the function")
	qosClass := invokeCmd.String("qosclass", "", "QoS class (optional)")
	qosMaxRespT := invokeCmd.Float64("qosrespt", -1.0, "Max. response time (optional)")
	invokeCmd.Var(&params, "param", "Function parameter: <name>:<value>")
	invokeCmd.Parse(os.Args[2:])

	if len(*funcName) < 1 {
		fmt.Printf("Invalid function name.\n")
		ExitWithUsage()
	}

	// Prepare request
	request := function.InvocationRequest{Params: params, QoSClass: api.DecodePriority(*qosClass), QoSMaxRespT: *qosMaxRespT}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		ExitWithUsage()
	}

	// Send invocation request
	url := fmt.Sprintf("http://%s:%d/invoke/%s", ServerConfig.Host, ServerConfig.Port, *funcName)
	resp, err := utils.PostJson(url, invocationBody)
	if err != nil {
		fmt.Printf("Invocation failed: %v", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}
