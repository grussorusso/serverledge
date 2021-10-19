package containers

import (
	"log"

	"github.com/grussorusso/serverledge/pkg/functions"
)

type ContainerID = string

func GetWarmContainer (f *functions.Function) (contID ContainerID, found bool) {
	found = false
	// TODO: check if we have a warm container for f
	// TODO: synchronization needed
	return contID, found
}

func WarmStart (r *functions.Request, c ContainerID) error {
	log.Printf("Starting warm container %v", c)
	return nil
}

func ColdStart (r *functions.Request) error {
	log.Printf("Starting new container for %v", r.Fun)
	// TODO: choose image based on runtime and set command and memory
	image := "alpine"
	_, err := cf.Create(image, &ContainerOptions{})
	return err
}
