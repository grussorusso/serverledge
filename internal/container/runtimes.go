package container

// RuntimeInfo contains information about a supported function runtime env.
type RuntimeInfo struct {
	Image         string
	InvocationCmd []string
}

const CUSTOM_RUNTIME = "custom"

var refreshedImages = map[string]bool{}

var RuntimeToInfo = map[string]RuntimeInfo{
	//TODO reset container repo
	"python310":  RuntimeInfo{"ferrarally/serverledge-python310", []string{"python", "/entrypoint.py"}},
	"nodejs17":   RuntimeInfo{"ferrarally/serverledge-nodejs17", []string{"node", "/entrypoint.js"}},
	"nodejs17ng": RuntimeInfo{"ferrarally/serverledge-nodejs17ng", []string{}},
}
