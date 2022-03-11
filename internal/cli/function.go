package cli

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
)

func Create(funcName string, runtime string, customImage string, src string, handler string, memory int64, cpuDemand float64) {
	if funcName == "" || runtime == "" {
		ExitWithUsage()
	}

	if runtime == "custom" && customImage == "" {
		ExitWithUsage()
	} else if runtime != "custom" && src == "" {
		ExitWithUsage()
	}

	var encoded string
	if runtime != "custom" {
		srcContent, err := readSourcesAsTar(src)
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
		ExitWithUsage()
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
		tarFileName = file.Name()
	} else {
		// this is already a tar file
		tarFileName = srcPath
	}

	return ioutil.ReadFile(tarFileName)
}

func Delete() {
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	funcName := deleteCmd.String("function", "", "name of the function")
	deleteCmd.Parse(os.Args[2:])

	request := function.Function{Name: *funcName}
	requestBody, err := json.Marshal(request)
	if err != nil {
		ExitWithUsage()
	}

	url := fmt.Sprintf("http://%s:%d/delete", ServerConfig.Host, ServerConfig.Port)
	resp, err := utils.PostJson(url, requestBody)
	if err != nil {
		fmt.Printf("Deletion request failed: %v\n", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}

func List() {
	url := fmt.Sprintf("http://%s:%d/function", ServerConfig.Host, ServerConfig.Port)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("List request failed: %v\n", err)
		os.Exit(2)
	}
	utils.PrintJsonResponse(resp.Body)
}
