![ServerlEdge](docs/logo.png)

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

## Compilation

Build the project from sources:

	$ make

## Running locally

You need an **etcd** server to be up and running. To quickly start a local
server:

	$ ./scripts/start-etcd.sh

Start the server:

	$ bin/serverledge

Create a function `func` from example code:

	$ bin/serverledge-cli create -function func -memory 600 -src examples/hello.py -runtime python310 -handler "hello.handler"

Invoke a function `func` with parameters `a=2` and `b=3`:

	$ bin/serverledge-cli invoke -function func -param "a:2" -param "b:3" 

You can optionally specify a QoS class name and a maximum requested response
time:

	$ bin/serverledge-cli invoke -function func -param ... -qosclass <class> -qosrespt <respt>

Get Server Status:
 
    $ bin/serverledge-cli status -host {host ip-address} -port {specific port}


To shutdown the etcd server:

	$ ./scripts/stop-etcd.sh

## Configuration

You can provide a configuration file using YAML or TOML syntax. Depending on the
chosen format, the default file name will be `serverledge-conf.yaml` or
`serverledge-conf.toml`. The file can be either placed in `/etc/serverledge`,
in the user `$HOME` directory, or in the working directory where the server is
started.

Alternatively, you can indicate a specific configuration file when starting the 
server:

	$ bin/serverledge <config file>

Supported configuration keys are defined in `internal/config/keys.go`.

## Using the CLI with a remote server

You can append the options `-host HOST` and `-port PORT` to specify the server
host and port. Alternatively, you can set the environment variables
`SERVERLEDGE_HOST` and/or `SERVERLEDGE_PORT`, which are read by the client.


## Writing functions

### Python

Available runtime: `python310` (Python 3.10)

	def handler_fun (context, params):
		return "..."

Specify the handler as `<module_name>.<function_name>` (e.g., `myfile.handler_fun`).
An example is given in `examples/hello.py`.

### NodeJS

Available runtime: `nodejs17` (NodeJS 17)

	function handler_fun (context, params) {
		return "..."
	}
	
	module.exports = handler_fun // this is mandatory!

Specify the handler as `<script_file_name>.js` (e.g., `myfile.js`).
An example is given in `examples/hello.js`.
