package container

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/grussorusso/serverledge/internal/config"
	//	"github.com/docker/docker/pkg/stdcopy"
)

type DockerFactory struct {
	cli *client.Client
	ctx context.Context
}

func InitDockerContainerFactory() *DockerFactory {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	dockerFact := &DockerFactory{cli, ctx}
	cf = dockerFact
	return dockerFact
}

func (cf *DockerFactory) Create(image string, opts *ContainerOptions) (ContainerID, error) {
	if !cf.HasImage(image) {
		log.Printf("Pulling image: %s", image)
		pullResp, err := cf.cli.ImagePull(cf.ctx, image, types.ImagePullOptions{})
		if err != nil {
			log.Printf("Could not pull image: %s", image)
			// we do not return here, as a stale copy of the image
			// could still be available locally
		} else {
			defer pullResp.Close()
			// This seems to be necessary to wait for the image to be pulled:
			io.Copy(ioutil.Discard, pullResp)
			log.Printf("Pulled image: %s", image)
			refreshedImages[image] = true
		}
	}

	resp, err := cf.cli.ContainerCreate(cf.ctx, &container.Config{
		Image: image,
		Cmd:   opts.Cmd,
		Env:   opts.Env,
		Tty:   false,
	}, &container.HostConfig{
		Resources: container.Resources{Memory: opts.MemoryMB * 1048576}, // convert to bytes
	}, nil, nil, "")

	return resp.ID, err
}

func (cf *DockerFactory) CopyToContainer(contID ContainerID, content io.Reader, destPath string) error {
	return cf.cli.CopyToContainer(cf.ctx, contID, destPath, content, types.CopyToContainerOptions{})
}

func (cf *DockerFactory) Start(contID ContainerID) error {
	if err := cf.cli.ContainerStart(cf.ctx, contID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	return nil
}

func (cf *DockerFactory) Destroy(contID ContainerID) error {
	// force set to true causes running container to be killed (and then
	// removed)
	return cf.cli.ContainerRemove(cf.ctx, contID, types.ContainerRemoveOptions{Force: true})
}

func (cf *DockerFactory) HasImage(image string) bool {
	// TODO: we should try using cf.cli.ImageList(...)
	cmd := fmt.Sprintf("docker images %s | grep -vF REPOSITORY", image)
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return false
	}

	// We have the image, but we may need to refresh it
	if config.GetBool(config.FACTORY_REFRESH_IMAGES, false) {
		if refreshed, ok := refreshedImages[image]; !ok || !refreshed {
			return false
		}
	}
	return true
}

func (cf *DockerFactory) GetIPAddress(contID ContainerID) (string, error) {
	contJson, err := cf.cli.ContainerInspect(cf.ctx, contID)
	if err != nil {
		return "", err
	}
	return contJson.NetworkSettings.IPAddress, nil
}

func (cf *DockerFactory) GetMemoryMB(contID ContainerID) (int64, error) {
	contJson, err := cf.cli.ContainerInspect(cf.ctx, contID)
	if err != nil {
		return -1, err
	}
	return contJson.HostConfig.Memory / 1048576, nil
}

/* Experimental feature: checkpoints a container
Container checkpoint Docker API does not support the --tcp-established flag yet.
It is recommended to use Podman in order to migrate a container.*/
func (cf *DockerFactory) CheckpointContainer(contID ContainerID, archiveName string) error {
	err := cf.cli.CheckpointCreate(cf.ctx, contID, types.CheckpointCreateOptions{CheckpointID: contID, CheckpointDir: archiveName})
	if err != nil {
		log.Printf("The container %s could not be checkpointed: %v", contID, err)
	}
	/* TODO: before returning, this function has to create a tar archive from the checkpoint located
	in /var/lib/docker/containers/container_ID/checkpoints, and put it in the current working directory.*/
	return err
}

/* Experimental feature: restores a container
Container checkpoint Docker API does not support the --tcp-established flag yet.
It is recommended to use Podman in order to migrate a container.*/
func (cf *DockerFactory) RestoreContainer(contID ContainerID, archiveName string) (string, error) {
	/* TODO: before restoring, this function has to create the container without starting it.
	Then it has to extract the local tar archive and move it into /var/lib/docker/containers/container_ID/checkpoints*/
	err := cf.cli.ContainerStart(cf.ctx, contID, types.ContainerStartOptions{CheckpointID: contID, CheckpointDir: archiveName})
	if err != nil {
		log.Printf("The container %s could not be restored: %v", contID, err)
	}
	return contID, err
}
