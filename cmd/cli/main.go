package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/grussorusso/serverledge/internal/cli"
	"github.com/grussorusso/serverledge/internal/config"
)

func main() {
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

	cli.Init()
}
