BIN=bin

all: serverledge executor serverledge-cli lb

serverledge:
	CGO_ENABLED=1 GOOS=linux go build -o $(BIN)/$@ cmd/$@/main.go
lb:
	CGO_ENABLED=1 GOOS=linux go build -o $(BIN)/$@ cmd/$@/main.go

serverledge-cli:
	CGO_ENABLED=1 GOOS=linux go build -o $(BIN)/$@ cmd/cli/main.go

executor:
	CGO_ENABLED=0 GOOS=linux go build -o $(BIN)/$@ cmd/$@/executor.go

DOCKERHUB_USER=grussorusso
CONTAINER_MANAGER=$(manager)

images:  image-python310 image-nodejs17ng image-base image-custom-python
image-python310:
	${CONTAINER_MANAGER} build -t $(DOCKERHUB_USER)/serverledge-python310 -f images/python310/Dockerfile .
image-base:
	${CONTAINER_MANAGER} build -t $(DOCKERHUB_USER)/serverledge-base -f images/base-alpine/Dockerfile .
image-nodejs17ng:
	${CONTAINER_MANAGER} build -t $(DOCKERHUB_USER)/serverledge-nodejs17ng -f images/nodejs17ng/Dockerfile .
image-custom-python:
	${CONTAINER_MANAGER} build -t docker.io/mferretti1997/serverledge-custom-python -f images/custom-python/Dockerfile .

push-images:
	${CONTAINER_MANAGER} push $(DOCKERHUB_USER)/serverledge-python310
	${CONTAINER_MANAGER} push $(DOCKERHUB_USER)/serverledge-base
	${CONTAINER_MANAGER} push $(DOCKERHUB_USER)/serverledge-nodejs17ng

test:
	go test -v ./...

.PHONY: serverledge serverledge-cli lb executor test images

	
