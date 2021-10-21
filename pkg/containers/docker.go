package containers

import (
	"context"
	"io"
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

func (cf *DockerFactory) Create (image string, opts *ContainerOptions) (ContainerID, error) {
	reader, err := cf.cli.ImagePull(cf.ctx, image, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	io.Copy(os.Stdout, reader) // TODO

	resp, err := cf.cli.ContainerCreate(cf.ctx, &container.Config{
		Image: image,
		Cmd:   opts.Cmd,
		Env: opts.Env,
		Tty:   false,
	}, nil, nil, nil, "")

	return resp.ID, err
}

func (cf *DockerFactory) CopyToContainer (contID ContainerID, content io.Reader, destPath string)  error {
	return cf.cli.CopyToContainer(cf.ctx, contID, destPath,  content, types.CopyToContainerOptions{})
}

func (cf *DockerFactory) Start (contID ContainerID) error {
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

