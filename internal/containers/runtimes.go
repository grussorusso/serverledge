package containers

//RuntimeInfo contains information about a supported function runtime env.
type RuntimeInfo struct {
	Image         string
	InvocationCmd []string
}

var refreshedImages = map[string]bool{}

var runtimeToInfo = map[string]RuntimeInfo{
	"python310": RuntimeInfo{"grussorusso/serverledge-python310", []string{"python", "/entrypoint.py"}},
}
