BIN=bin

all: serverledge executor serverledge-cli

serverledge:
	GOOS=linux go build -o $(BIN)/$@ cmd/$@/main.go

serverledge-cli:
	GOOS=linux go build -o $(BIN)/$@ cmd/cli/main.go

executor:
	CGO_ENABLED=0 GOOS=linux go build -o $(BIN)/$@ cmd/$@/executor.go

DOCKERHUB_USER=grussorusso
images: image-nodejs17 image-python310

image-python310:
	docker build -t $(DOCKERHUB_USER)/serverledge-python310 -f images/python310/Dockerfile .
image-nodejs17:
	docker build -t $(DOCKERHUB_USER)/serverledge-nodejs17 -f images/nodejs17/Dockerfile .

push-images:
	docker push $(DOCKERHUB_USER)/serverledge-python310
	docker push $(DOCKERHUB_USER)/serverledge-nodejs17

test:
	go test -v ./...

.PHONY: serverledge serverledge-cli executor test images

	
