package containers

import (
	"log"
	"os"

	"github.com/grussorusso/serverledge/pkg/functions"
)

type ContainerID = string

func GetWarmContainer (f *functions.Function) (contID ContainerID, found bool) {
	found = false
	// TODO: check if we have a warm container for f
	// TODO: synchronization needed
	return contID, found
}

func WarmStart (r *functions.Request, c ContainerID) (string, error) {
	log.Printf("Starting warm container %v", c)
	return invoke(c, r)
}

func ColdStart (r *functions.Request) (string, error) {
	runtimeInfo := runtimeToInfo[r.Fun.Runtime]
	image := runtimeInfo.Image
	cmd := runtimeInfo.Command
	log.Printf("Starting new container for %s (image: %s)", r.Fun, image)

	// TODO: set memory

	opts := &ContainerOptions {
		Cmd: cmd,
	}
	contID, err := cf.Create(image, opts)
	if err != nil {
		log.Printf("Failed container creation: %v", err)
		return "", err
	}

	content, ferr := os.Open(r.Fun.SourceTarURL) // TODO: HTTP
	defer content.Close()
	if ferr != nil {
		log.Fatalf("Reading failed: %v", ferr)
	}
	err = cf.CopyToContainer(contID, content, "/app/")
	if err != nil {
		log.Fatalf("Copy failed: %v", err)
	}

	err = cf.Start(contID)
	if err != nil {
		log.Fatalf("Starting container failed: %v", err)
	}

	return invoke(contID, r)
}

func invoke (contID string, r *functions.Request) (string, error) {
	//TODO: send request to executor within container
	return "", nil
}
