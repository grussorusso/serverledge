package container

// RuntimeInfo contains information about a supported function runtime env.
type RuntimeInfo struct {
	Image         string
	InvocationCmd []string
}

const CUSTOM_RUNTIME = "custom"

var refreshedImages = map[string]bool{}

var RuntimeToInfo = map[string]RuntimeInfo{
	"python310": RuntimeInfo{"grussorusso/serverledge-python310", []string{"python", "/entrypoint.py"}},
	// "nodejs17":   RuntimeInfo{"grussorusso/serverledge-nodejs17", []string{"node", "/entrypoint.js"}},// TODO: it doesn't work!
	"nodejs17ng": RuntimeInfo{"grussorusso/serverledge-nodejs17ng", []string{}},
}
