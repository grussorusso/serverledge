package container

//RuntimeInfo contains information about a supported function runtime env.
type RuntimeInfo struct {
	Image         string
	InvocationCmd []string
}

const CUSTOM_RUNTIME = "custom"

var refreshedImages = map[string]bool{}

// Podman requires the prefix 'docker.io' in order to pull from DockerHub
var RuntimeToInfo = map[string]RuntimeInfo{
	"python310":  {"docker.io/grussorusso/serverledge-python310", []string{"python", "/entrypoint.py"}},
	"nodejs17":   {"docker.io/grussorusso/serverledge-nodejs17", []string{"node", "/entrypoint.js"}},
	"nodejs17ng": {"docker.io/grussorusso/serverledge-nodejs17ng", []string{}},
}
