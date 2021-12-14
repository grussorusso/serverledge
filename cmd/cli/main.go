package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/grussorusso/serverledge/internal/api"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/functions"
	"github.com/grussorusso/serverledge/utils"
)

var serverConfig config.RemoteServerConf

func exitWithUsage() {
	fmt.Println("expected a subcommanda: 'invoke', 'create', 'list'")
	os.Exit(1)
}

func main() {
	config.ReadConfiguration("")

	// Set defaults
	serverConfig.Host = "127.0.0.1"
	serverConfig.Port = config.GetInt("api.port", 1323)

	// Parse general configuration
	flag.IntVar(&serverConfig.Port, "port", serverConfig.Port, "port for remote connection")
	flag.StringVar(&serverConfig.Host, "host", serverConfig.Host, "host for remote connection")
	flag.Parse()

	if len(os.Args) < 2 {
		exitWithUsage()
	}

	switch os.Args[1] {

	case "invoke":
		invoke()
	case "create":
		create()
	case "list":
		list()
	default:
		exitWithUsage()
	}
}

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

func invoke() {
	var params paramsFlags = make(map[string]string)

	invokeCmd := flag.NewFlagSet("invoke", flag.ExitOnError)
	funcName := invokeCmd.String("function", "", "name of the function")
	qosClass := invokeCmd.String("qosclass", "", "QoS class (optional)")
	qosMaxRespT := invokeCmd.Float64("qosrespt", -1.0, "Max. response time (optional)")
	invokeCmd.Var(&params, "param", "Function parameter: <name>:<value>")
	invokeCmd.Parse(os.Args[2:])

	if len(*funcName) < 1 {
		fmt.Printf("Invalid function name.\n")
		exitWithUsage()
	}

	// Prepare request
	request := api.FunctionInvocationRequest{params, *qosClass, *qosMaxRespT}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		exitWithUsage()
	}

	// Send invocation request
	url := fmt.Sprintf("http://%s:%d/invoke/%s", serverConfig.Host, serverConfig.Port, *funcName)
	resp, err := postJson(url, invocationBody)
	if err != nil {
		fmt.Printf("Invocation failed: %v", err)
		os.Exit(2)
	}
	printJsonResponse(resp.Body)
}

func readSourcesAsTar(srcPath string) ([]byte, error) {
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
		fmt.Printf("Created temporary archive: %s", file.Name())
		tarFileName = file.Name()
	} else {
		// this is already a tar file
		tarFileName = srcPath
	}

	return ioutil.ReadFile(tarFileName)
}

func create() {
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	funcName := createCmd.String("function", "", "name of the function")
	runtime := createCmd.String("runtime", "python38", "runtime for the function")
	handler := createCmd.String("handler", "", "function handler")
	memory := createCmd.Int("memory", 128, "max memory in MB for the function")
	src := createCmd.String("src", "", "source the function (single file, directory or TAR archive)")
	createCmd.Parse(os.Args[2:])

	srcContent, err := readSourcesAsTar(*src)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(3)
	}
	encoded := base64.StdEncoding.EncodeToString(srcContent)

	request := functions.Function{Name: *funcName, Handler: *handler, Runtime: *runtime, MemoryMB: int64(*memory), TarFunctionCode: encoded}
	requestBody, err := json.Marshal(request)
	if err != nil {
		exitWithUsage()
	}

	url := fmt.Sprintf("http://%s:%d/create", serverConfig.Host, serverConfig.Port)
	resp, err := postJson(url, requestBody)
	if err != nil {
		// TODO: check returned error code
		fmt.Printf("Creation request failed: %v\n", err)
		os.Exit(2)
	}
	printJsonResponse(resp.Body)
}

func postJson(url string, body []byte) (*http.Response, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("Server response: %v", resp.Status)
	}
	return resp, nil
}

func printJsonResponse(resp io.ReadCloser) {
	defer resp.Close()
	body, _ := ioutil.ReadAll(resp)

	// print indented JSON
	var out bytes.Buffer
	json.Indent(&out, body, "", "\t")
	out.WriteTo(os.Stdout)
}

func list() {
	url := fmt.Sprintf("http://%s:%d/functions", serverConfig.Host, serverConfig.Port)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("List request failed: %v\n", err)
		os.Exit(2)
	}
	printJsonResponse(resp.Body)
}
