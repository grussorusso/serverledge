BIN=bin

all: serverledge executor

serverledge:
	GOOS=linux go build -o $(BIN)/$@ cmd/$@/main.go

executor:
	CGO_ENABLED=0 GOOS=linux go build -o $(BIN)/$@ cmd/$@/executor.go

	
