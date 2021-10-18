BIN=bin

all: serverledge

serverledge:
	GOOS=linux go build -o $(BIN)/$@ cmd/$@/main.go

	
