package containers

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

type DockerFactory struct {
	cli *client.Client
	ctx context.Context
}

func InitDockerContainerFactory() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	cf = &DockerFactory{cli, ctx}
}

func (cf *DockerFactory) Create (image string, cmd []string, opts *ContainerOptions) (ContainerID, error) {
	reader, err := cf.cli.ImagePull(cf.ctx, image, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	io.Copy(os.Stdout, reader) // TODO

	resp, err := cf.cli.ContainerCreate(cf.ctx, &container.Config{
		Image: image,
		Cmd:   cmd,
		Tty:   false,
	}, nil, nil, nil, "")

	return resp.ID, err
}

func (cf *DockerFactory) Start (contID ContainerID) error {
	// TODO: test copy
	content, ferr := os.Open("/tmp/prova.tar")
	defer content.Close()
	if ferr != nil {
		log.Fatalf("Reading failed: %v", ferr)
	}
	if err := cf.cli.CopyToContainer(cf.ctx, contID, "/",  content, types.CopyToContainerOptions{}); err != nil {
		log.Fatalf("Copy failed: %v", err)
	}

	if err := cf.cli.ContainerStart(cf.ctx, contID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	statusCh, errCh := cf.cli.ContainerWait(cf.ctx, contID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-statusCh:
	}

	out, err := cf.cli.ContainerLogs(cf.ctx, contID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return err
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return nil
}

