package containers

type RuntimeInfo struct {
	Image string
	InvocationCmd []string
}

var runtimeToInfo = map[string]RuntimeInfo{
	"python310": RuntimeInfo{"grussorusso/serverledge-python310", []string{"python","/entrypoint.py"}},
}
