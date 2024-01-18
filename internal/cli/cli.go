package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/fc"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/grussorusso/serverledge/internal/client"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
	"github.com/spf13/cobra"
)

var ServerConfig config.RemoteServerConf

var rootCmd = &cobra.Command{
	Use:   "serverledge-cli",
	Short: "CLI utility for Serverledge",
	Long:  `CLI utility to interact with a Serverledge FaaS platform.`,
}

var invokeCmd = &cobra.Command{
	Use:   "invoke",
	Short: "Invokes a function",
	Run:   invoke,
}

var pollCmd = &cobra.Command{
	Use:   "poll",
	Short: "Polls the result of an asynchronous invocation",
	Run:   poll,
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Registers a new function",
	Run:   create,
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Deletes a function",
	Run:   deleteFunction,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists registered functions",
	Run:   listFunctions,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Prints status information about the system",
	Run:   getStatus,
}

// ========== FUNCTION COMPOSITION ===========

var compCreateCmd = &cobra.Command{
	Use:   "compose",
	Short: "Registers a new function composition",
	Run:   createComposition,
}

var compDeleteCmd = &cobra.Command{
	Use:   "uncompose",
	Short: "Deletes a function composition and optionally the associated functions",
	Run:   deleteComposition,
}

var compListCmd = &cobra.Command{
	Use:   "fc",
	Short: "Lists registered function compositions",
	Run:   listFunctionCompositions,
}

var compInvokeCmd = &cobra.Command{
	Use:   "play",
	Short: "Invokes a function composition",
	Run:   invokeFunctionComposition,
}

var compPollCmd = &cobra.Command{
	Use:   "peek",
	Short: "Polls the result of an asynchronous function composition invocation",
	Run:   pollFunctionComposition,
}

var compName, funcName, runtime, handler, customImage, src, qosClass, yamlSrc, jsonSrc string
var requestId string
var memory int64
var cpuDemand, qosMaxRespT float64
var params []string
var paramsFile string
var asyncInvocation bool
var verbose bool
var removeFnOnDelete bool

func Init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&ServerConfig.Host, "host", "H", ServerConfig.Host, "remote Serverledge host")
	rootCmd.PersistentFlags().IntVarP(&ServerConfig.Port, "port", "P", ServerConfig.Port, "remote Serverledge port")

	rootCmd.AddCommand(invokeCmd)
	invokeCmd.Flags().StringVarP(&funcName, "function", "f", "", "name of the function")
	invokeCmd.Flags().Float64VarP(&qosMaxRespT, "resptime", "", -1.0, "Max. response time (optional)")
	invokeCmd.Flags().StringVarP(&qosClass, "class", "c", "", "QoS class (optional)")
	invokeCmd.Flags().StringSliceVarP(&params, "param", "p", nil, "Function parameter: <name>:<value>")
	invokeCmd.Flags().StringVarP(&paramsFile, "params_file", "j", "", "File containing parameters (JSON)")
	invokeCmd.Flags().BoolVarP(&asyncInvocation, "async", "a", false, "Asynchronous invocation")

	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&funcName, "function", "f", "", "name of the function")
	createCmd.Flags().StringVarP(&runtime, "runtime", "", "python38", "runtime for the function")
	createCmd.Flags().StringVarP(&handler, "handler", "", "", "function handler (runtime specific)")
	createCmd.Flags().Int64VarP(&memory, "memory", "", 128, "memory (in MB) for the function")
	createCmd.Flags().Float64VarP(&cpuDemand, "cpu", "", 0.0, "estimated CPU demand for the function (1.0 = 1 core)")
	createCmd.Flags().StringVarP(&src, "src", "", "", "source for the function (single file, directory or TAR archive) (not necessary for runtime==custom)")
	createCmd.Flags().StringVarP(&customImage, "custom_image", "", "", "custom container image (only if runtime == 'custom')")

	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringVarP(&funcName, "function", "f", "", "name of the function")

	rootCmd.AddCommand(listCmd)

	rootCmd.AddCommand(statusCmd)

	rootCmd.AddCommand(pollCmd)
	pollCmd.Flags().StringVarP(&requestId, "request", "", "", "ID of the async request")

	// Function composition

	rootCmd.AddCommand(compInvokeCmd)
	compInvokeCmd.Flags().StringVarP(&compName, "function-composition", "f", "", "name of the function composition")
	compInvokeCmd.Flags().Float64VarP(&qosMaxRespT, "resptime", "r", -1.0, "Max. response time (optional)")
	compInvokeCmd.Flags().StringVarP(&qosClass, "class", "c", "", "QoS class (optional)")
	compInvokeCmd.Flags().StringSliceVarP(&params, "param", "p", nil, "Composition parameter: <name>:<value>")
	compInvokeCmd.Flags().StringVarP(&paramsFile, "params_file", "j", "", "File containing parameters (JSON) for composition")
	compInvokeCmd.Flags().BoolVarP(&asyncInvocation, "async", "a", false, "Asynchronous composition invocation")

	rootCmd.AddCommand(compCreateCmd)
	compCreateCmd.Flags().StringVarP(&compName, "function-composition", "f", "", "name of the function")
	compCreateCmd.Flags().StringVarP(&yamlSrc, "afcl", "y", "", "source YAML file that defines for the function composition with AFCL syntax (if specified, 'aws' must not be set)")
	compCreateCmd.Flags().StringVarP(&jsonSrc, "aws", "j", "", "source JSON file  that defines for the function composition with AWS Step Function syntax (if specified, 'afcl' must not be set)")
	compCreateCmd.Flags().BoolVarP(&removeFnOnDelete, "remove-function-on-delete", "r", false, "when the function composition is deleted, if this flag is true, the associated function will also be deleted")

	rootCmd.AddCommand(compDeleteCmd)
	compDeleteCmd.Flags().StringVarP(&compName, "function-composition", "f", "", "name of the function composition")

	rootCmd.AddCommand(compListCmd)

	// TODO: maybe useless
	rootCmd.AddCommand(compPollCmd)
	compPollCmd.Flags().StringVarP(&requestId, "request", "", "", "ID of the async request")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func invoke(cmd *cobra.Command, args []string) {
	if len(funcName) < 1 {
		fmt.Printf("Invalid function name.\n")
		cmd.Help()
		os.Exit(1)
	}

	// Parse parameters
	paramsMap := make(map[string]interface{})

	// Parameters can be specified either via file ("--params_file") or via cli ("--param")
	if len(params) > 0 && len(paramsFile) > 0 {
		fmt.Println("Parameters must be specified using either --param OR --params_file")
		os.Exit(1)
	}
	if len(params) > 0 {
		for _, rawParam := range params {
			tokens := strings.Split(rawParam, ":")
			if len(tokens) < 2 {
				cmd.Help()
				return
			}
			paramsMap[tokens[0]] = strings.Join(tokens[1:], ":")
		}
	}
	if len(paramsFile) > 0 {
		jsonFile, err := os.Open(paramsFile)
		defer jsonFile.Close()
		byteValue, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(byteValue, &paramsMap)
		if err != nil {
			fmt.Printf("Could not parse JSON-encoded parameters from '%s'\n", paramsFile)
			os.Exit(1)
		}
	}

	// Prepare request
	request := client.InvocationRequest{
		Params:   paramsMap,
		QoSClass: api.DecodeServiceClass(qosClass),
		// QoSClass:        qosClass,
		QoSMaxRespT:     qosMaxRespT,
		CanDoOffloading: true,
		Async:           asyncInvocation}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		cmd.Help()
		os.Exit(1)
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

func create(cmd *cobra.Command, args []string) {
	if funcName == "" || runtime == "" {
		cmd.Help()
		os.Exit(1)
	}

	if runtime == "custom" && customImage == "" {
		cmd.Help()
		os.Exit(1)
	} else if runtime != "custom" && src == "" {
		cmd.Help()
		os.Exit(1)
	}

	var encoded string
	if runtime != "custom" {
		srcContent, err := ReadSourcesAsTar(src)
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(3)
		}
		encoded = base64.StdEncoding.EncodeToString(srcContent)
	} else {
		encoded = ""
	}

	request := function.Function{Name: funcName, Handler: handler,
		Runtime: runtime, MemoryMB: memory,
		CPUDemand:       cpuDemand,
		TarFunctionCode: encoded,
		CustomImage:     customImage,
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		cmd.Help()
		os.Exit(1)
	}

	url := fmt.Sprintf("http://%s:%d/create", ServerConfig.Host, ServerConfig.Port)
	resp, err := utils.PostJson(url, requestBody)
	if err != nil {
		// TODO: check returned error code
		fmt.Printf("Creation request failed: %v\n", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}

func ReadSourcesAsTar(srcPath string) ([]byte, error) {
	fileInfo, err := os.Stat(srcPath)
	if err != nil {
		return nil, fmt.Errorf("Missing source file")
	}

	var tarFileName string

	if fileInfo.IsDir() || !strings.HasSuffix(srcPath, ".tar") {
		file, err := ioutil.TempFile("/tmp", "serverledgesource")
		if err != nil {
			return nil, err
		}
		defer os.Remove(file.Name())

		utils.Tar(srcPath, file)
		tarFileName = file.Name()
	} else {
		// this is already a tar file
		tarFileName = srcPath
	}

	return ioutil.ReadFile(tarFileName)
}

func deleteFunction(cmd *cobra.Command, args []string) {
	request := function.Function{Name: funcName}
	requestBody, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(2)
	}

	url := fmt.Sprintf("http://%s:%d/delete", ServerConfig.Host, ServerConfig.Port)
	resp, err := utils.PostJson(url, requestBody)
	if err != nil {
		fmt.Printf("Deletion request failed: %v\n", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}

func listFunctions(cmd *cobra.Command, args []string) {
	url := fmt.Sprintf("http://%s:%d/function", ServerConfig.Host, ServerConfig.Port)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("List request failed: %v\n", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}

func getStatus(cmd *cobra.Command, args []string) {
	url := fmt.Sprintf("http://%s:%d/status", ServerConfig.Host, ServerConfig.Port)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Invocation failed: %v", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}

func poll(cmd *cobra.Command, args []string) {
	if len(requestId) < 1 {
		cmd.Help()
		os.Exit(1)
	}

	url := fmt.Sprintf("http://%s:%d/poll/%s", ServerConfig.Host, ServerConfig.Port, requestId)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Polling request failed: %v\n", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
	fmt.Println()
}

// ============== FUNTION COMPOSITION ===============

func invokeFunctionComposition(cmd *cobra.Command, args []string) {
	if len(compName) < 1 {
		fmt.Printf("Invalid composition name.\n")
		cmd.Help()
		os.Exit(1)
	}

	// Parse parameters
	paramsMap := make(map[string]interface{})

	// Parameters can be specified either via file ("--params_file") or via cli ("--param")
	if len(params) > 0 && len(paramsFile) > 0 {
		fmt.Println("Parameters must be specified using either --param OR --params_file")
		os.Exit(1)
	}
	if len(params) > 0 {
		for _, rawParam := range params {
			tokens := strings.Split(rawParam, ":")
			if len(tokens) < 2 {
				cmd.Help()
				return
			}
			paramsMap[tokens[0]] = strings.Join(tokens[1:], ":")
		}
	}
	if len(paramsFile) > 0 {
		jsonFile, err := os.Open(paramsFile)
		defer jsonFile.Close()
		byteValue, _ := io.ReadAll(jsonFile)
		err = json.Unmarshal(byteValue, &paramsMap)
		if err != nil {
			fmt.Printf("Could not parse JSON-encoded parameters from '%s'\n", paramsFile)
			os.Exit(1)
		}
	}

	// Prepare request // TODO: it's ok to reuse the same type that function invocation uses?
	request := client.InvocationRequest{
		Params:   paramsMap,
		QoSClass: api.DecodeServiceClass(qosClass),
		// QoSClass:        qosClass,
		QoSMaxRespT:     qosMaxRespT,
		CanDoOffloading: true,
		Async:           asyncInvocation}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		cmd.Help()
		os.Exit(1)
	}

	// Send invocation request
	url := fmt.Sprintf("http://%s:%d/play/%s", ServerConfig.Host, ServerConfig.Port, compName)
	resp, err := utils.PostJson(url, invocationBody)
	if err != nil {
		fmt.Println(err)
		if resp != nil {
			utils.PrintErrorResponse(resp.Body)
		}
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}

func createComposition(cmd *cobra.Command, args []string) {
	if compName == "" || (yamlSrc == "" && jsonSrc == "") {
		cmd.Help()
		os.Exit(1)
	}

	if yamlSrc != "" && jsonSrc != "" {
		cmd.Help()
		os.Exit(1)
	}

	var dag fc.Dag
	var funcSlice []*function.Function
	if yamlSrc != "" {
		dag1, funcs, err := ReadFromYAML(yamlSrc)
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(3)
		}
		dag = *dag1
		funcSlice = funcs
	} else if jsonSrc != "" {
		dag1, funcs, err := ReadFromJSON(jsonSrc)
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(3)
		}
		dag = *dag1
		funcSlice = funcs
	}
	// getting all functions. Todo: register all functions

	request := fc.NewFC(compName, dag, funcSlice, removeFnOnDelete)

	requestBody, err := json.Marshal(request)
	if err != nil {
		cmd.Help()
		os.Exit(1)
	}

	url := fmt.Sprintf("http://%s:%d/compose", ServerConfig.Host, ServerConfig.Port)
	resp, err := utils.PostJson(url, requestBody)
	if err != nil {
		// TODO: check returned error code
		fmt.Printf("Creation request failed: %v\n", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)

}

func deleteComposition(cmd *cobra.Command, args []string) {
	if compName == "" {
		cmd.Help()
		os.Exit(1)
	}
	request := fc.FunctionComposition{
		Name: compName,
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(2)
	}
	url := fmt.Sprintf("http://%s:%d/uncompose", ServerConfig.Host, ServerConfig.Port)
	resp, err := utils.PostJson(url, requestBody)
	if err != nil {
		fmt.Printf("Deletion request failed: %v\n", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}

func listFunctionCompositions(cmd *cobra.Command, args []string) {
	url := fmt.Sprintf("http://%s:%d/fc", ServerConfig.Host, ServerConfig.Port)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("List request failed: %v\n", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}

// TODO: maybe remove, we already have pollFunction
func pollFunctionComposition(cmd *cobra.Command, args []string) {
	if len(requestId) < 1 {
		cmd.Help()
		os.Exit(1)
	}

	url := fmt.Sprintf("http://%s:%d/poll/%s", ServerConfig.Host, ServerConfig.Port, requestId)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Polling request failed: %v\n", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}
