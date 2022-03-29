package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/grussorusso/serverledge/internal/cli"
	"github.com/grussorusso/serverledge/internal/config"
)

func main() {
	var funcName, runtime, handler, customImage, src, qosClass string
	var memory int64
	var cpuDemand, qosMaxRespT float64
	var params cli.ParamsFlags = make(map[string]string)
	config.ReadConfiguration("")

	// Set defaults
	cli.ServerConfig.Host = "127.0.0.1"
	cli.ServerConfig.Port = config.GetInt("api.port", 1323)

	// Check for environment variables
	if envHost, ok := os.LookupEnv("SERVERLEDGE_HOST"); ok {
		cli.ServerConfig.Host = envHost
	}
	if envPort, ok := os.LookupEnv("SERVERLEDGE_PORT"); ok {
		if iPort, err := strconv.Atoi(envPort); err == nil {
			cli.ServerConfig.Port = iPort
		} else {
			fmt.Errorf("Invalid port number: %s\n", envPort)
		}
	}

	command := ""
	// Parse general configuration
	flag.IntVar(&cli.ServerConfig.Port, "port", cli.ServerConfig.Port, "port for remote connection")
	flag.StringVar(&cli.ServerConfig.Host, "host", cli.ServerConfig.Host, "host for remote connection")
	flag.StringVar(&command, "cmd", command, "command to execute")
	//parse parameters for function creation
	flag.StringVar(&funcName, "function", "", "name of the function")
	flag.StringVar(&runtime, "runtime", "python38", "runtime for the function")
	flag.StringVar(&handler, "handler", "", "function handler")
	flag.StringVar(&customImage, "custom_image", "", "custom container image (only if runtime is 'custom')")
	flag.Int64Var(&memory, "memory", 128, "max memory in MB for the function")
	flag.Float64Var(&cpuDemand, "cpu", 0.0, "estimated CPU demand for the function (e.g., 1.0 = 1 core)")
	flag.StringVar(&src, "src", "", "source the function (single file, directory or TAR archive)")
	//parameters for function invocation
	flag.Float64Var(&qosMaxRespT, "qosrespt", -1.0, "Max. response time (optional)")
	flag.StringVar(&qosClass, "qosclass", "", "QoS class (optional)")
	flag.Var(&params, "param", "Function parameter: <name>:<value>")
	flag.Parse()

	if len(os.Args) < 2 {
		cli.ExitWithUsage()
	}

	switch {

	case command == "invoke":
		cli.Invoke(funcName, qosClass, qosMaxRespT, params)
	case command == "create":
		cli.Create(funcName, runtime, customImage, src, handler, memory, cpuDemand)
	case command == "delete":
		cli.Delete()
	case command == "list":
		cli.List()
	case command == "status":
		cli.GetStatus()
	default:
		cli.ExitWithUsage()
	}
}
