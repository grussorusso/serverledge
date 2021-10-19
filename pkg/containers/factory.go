package containers

type ContainerOptions struct {
}

type Factory interface {
	Create(string, *ContainerOptions) (ContainerID, error)
	Start(ContainerID, []string)  error
}

var cf Factory


