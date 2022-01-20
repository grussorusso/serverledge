BIN=bin

all: serverledge executor serverledge-cli

serverledge:
	GOOS=linux go build -o $(BIN)/$@ cmd/$@/main.go

serverledge-cli:
	GOOS=linux go build -o $(BIN)/$@ cmd/cli/main.go

executor:
	CGO_ENABLED=0 GOOS=linux go build -o $(BIN)/$@ cmd/$@/executor.go

DOCKERHUB_USER=grussorusso
images:
	docker build -t $(DOCKERHUB_USER)/serverledge-python310 -f images/python310/Dockerfile .
	docker build -t $(DOCKERHUB_USER)/serverledge-nodejs17 -f images/nodejs17/Dockerfile .


test:
	go test -v ./...

.PHONY: serverledge serverledge-cli executor test images

	
