package container

import "github.com/grussorusso/serverledge/internal/config"

//RuntimeInfo contains information about a supported function runtime env.
type RuntimeInfo struct {
	Image         string
	InvocationCmd []string
}

const CUSTOM_RUNTIME = "custom"

var refreshedImages = map[string]bool{}

var RuntimeToInfo = getRuntimeInfo()

// Podman requires the prefix 'docker.io' in order to pull from DockerHub
func getRuntimeInfo() map[string]RuntimeInfo {
	config.ReadConfiguration(config.DefaultConfigFileName)
	containerManager := config.GetString(config.DEFAULT_CONTAINER_MANAGER, "podman")
	prefix := ""
	if containerManager == "podman" {
		prefix = "docker.io/"
	}
	return map[string]RuntimeInfo{
		"python310":  {prefix + "grussorusso/serverledge-python310", []string{"python", "/entrypoint.py"}},
		"nodejs17":   {prefix + "grussorusso/serverledge-nodejs17", []string{"node", "/entrypoint.js"}},
		"nodejs17ng": {prefix + "grussorusso/serverledge-nodejs17ng", []string{}},
	}
}
