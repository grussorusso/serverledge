package containers

import (
	"fmt"
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

func WarmStart (r *functions.Request, c ContainerID) error {
	log.Printf("Starting warm container %v", c)
	return nil
}

func ColdStart (r *functions.Request) error {
	image := runtimeToImage[r.Fun.Runtime]
	log.Printf("Starting new container for %s (image: %s)", r.Fun, image)

	// TODO: set command and memory

	env := make([]string, 2)
	env[0] = "HANDLER_DIR=/app/"
	env[1] = fmt.Sprintf("HANDLER=%s", r.Fun.Handler)

	cmd := []string{"python", "/entrypoint.py"} // TODO use runtimeToCmd map
	opts := &ContainerOptions {
		Cmd: cmd,
		Env: env,
	}
	contID, err := cf.Create(image, opts)
	if err != nil {
		log.Printf("Failed container creation: %v", err)
		return err
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

	return err
}
