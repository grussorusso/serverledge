package containers

import "io"

type ContainerOptions struct {
	Cmd []string
	Env []string
	MemoryMB int32
}

type Factory interface {
	Create(string, *ContainerOptions) (ContainerID, error)
	CopyToContainer (ContainerID, io.Reader, string) error
	Start(ContainerID)  error
	GetIPAddress(ContainerID) (string, error)
}

var cf Factory


