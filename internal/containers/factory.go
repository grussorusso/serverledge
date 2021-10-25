package containers

import "io"

//A Factory to create and manage containers.
type Factory interface {
	Create(string, *ContainerOptions) (ContainerID, error)
	CopyToContainer(ContainerID, io.Reader, string) error
	Start(ContainerID) error
	GetIPAddress(ContainerID) (string, error)
}

//ContainerOptions contains options for container creation.
type ContainerOptions struct {
	Cmd      []string
	Env      []string
	MemoryMB int32
}

type ContainerID = string

// cf is the container factory for the node
var cf Factory
