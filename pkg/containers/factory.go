package containers

type ContainerOptions struct {
}

type Factory interface {
	Create(string, []string, *ContainerOptions) (ContainerID, error)
	Start(ContainerID)  error
}

var cf Factory


