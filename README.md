# ServerlEdge #

ServerlEdge is a Function-as-a-Service (FaaS) platform specifically designed to
work in Edge/Fog environments.

## Architecture

ServerlEdge currently relies on a server component that executes functions 
locally. Functions are executed within containers. Each container is equipped
with an **Executor** component, which is a simple HTTP server. The Executor
listen for requests on port 8080. When a function must be invoked on a
container, a POST request is sent to the Executor with the invocation
parameters. The Executor response will contain the invocation result.

*... to be expanded ...*

## Building and Running

Build the project from sources:

	$ make

Start the server:

	$ bin/serverledge

Invoke a function for testing:

	$ curl -d '{"key1":"value1", "key2":"value2"}' -H "Content-Type: application/json" -X POST 127.0.0.1:1323/invoke/func

## Configuration

You can provide a configuration file using YAML or TOML syntax. Depending on the
chosen format, the file name will be `serverledge-conf.yaml` or
`serverledge-conf.toml`. The file can be either placed in `/etc/serverledge`,
in the user `$HOME` directory, or in the working directory where the server is
started.

## Organization of the Repository

	├── cmd
	│   ├── executor     # entrypoint for the Executor 
	│   └── serverledge  # entrypoint for the server
	├── images           # material to build runtime Docker images
	│   └── python310
	└── internal         # internal packages
	    ├── api          # main server API
	    ├── config       # configuration utilities
	    ├── containers   # container pools, containers management
	    ├── executor     # executor component
	    ├── functions    # function-related stuff
	    └── scheduling   # scheduling logic

