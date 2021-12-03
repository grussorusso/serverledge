BIN=bin

all: serverledge executor serverledge-cli

serverledge:
	GOOS=linux go build -o $(BIN)/$@ cmd/$@/main.go

serverledge-cli:
	GOOS=linux go build -o $(BIN)/$@ cmd/cli/main.go

executor:
	CGO_ENABLED=0 GOOS=linux go build -o $(BIN)/$@ cmd/$@/executor.go

	
