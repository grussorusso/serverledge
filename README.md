![ServerlEdge](docs/logo.png)

Serverledge is a Function-as-a-Service (FaaS) platform designed to
work in Edge-Cloud environments.

Serverledge allows user to define and execute functions across
distributed nodes located in Edge locations or in Cloud data centers.
Each Serverledge node serves function invocation requests using a local
container pool, so that functions are executed within containers for isolation purposes.
When Edge nodes are overloaded, Serverledge tries to offload computation
to neighbor Edge nodes or to the Cloud.

## Building from sources

1. Check that Golang is correctly installed on your machine.

1. Download a copy of the source code.

1. Build the project:

```
$ make
```

## Running (single node deployment)

You need an **etcd** server to run Serverledge. To quickly start a local
server:

	$ ./scripts/start-etcd.sh   # stop it with ./scripts/stop-etcd.sh

Start a Serverledge node:

	$ bin/serverledge

Register a function `func` from example code:

	$ bin/serverledge-cli create -f func --memory 600 --src examples/hello.py --runtime python310 --handler "hello.handler" 

Invoke `func` with arguments `a=2` and `b=3`:

	$ bin/serverledge-cli invoke -f func -p "a:2" -p "b:3"

For non-trivial inputs, it is recommended to specify function arguments through a
JSON file, instead of using the `-p` flag, as follows:

	$ bin/serverledge-cli invoke -f func --params_file input.json

where `input.json` may contain:

	{
		"a": 2,
		"b": 3
	}

Functions can be also invoked asynchronously using the `--async` flag:

	$ bin/serverledge-cli invoke -f func --async

The server will reply with a `requestID`, which can be used by the client to
poll for the execution result:

	$ bin/serverledge-cli poll --request <requestID>


## Distributed Deployment

[This repository](https://github.com/grussorusso/serverledge-deploy) provides an
Ansible playbook to deploy Serverledge in a distributed configuration.

In this case, you can instruct `serverledge-cli` to
connect to a node other than `localhost` or use a non-default port
by means of environment variables or command-line options:

- Use `--host <HOST>` (or `-H <HOST>`) and/or `--port <PORT>` (or, `-P <PORT>`)
to specify the server
host and port on the command line
- Alternatively, you can set the environment variables
`SERVERLEDGE_HOST` and/or `SERVERLEDGE_PORT`, which are read by the client.

Example:
 
    $ bin/serverledge-cli status -H <host ip-address> -P <port number>

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
An example is given in `examples/sieve.js`.

### Custom function runtimes

Follow [these instructions](./docs/custom_runtime.md).
