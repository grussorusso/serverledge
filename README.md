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

You need an **etcd** server to be up and running. To quickly start a local
server:

	$ ./scripts/start-etcd.sh

Start the server:

	$ bin/serverledge

> Note: in the following commands parameters order is not important

Create a function `func` from example code, you can optionally specify host and port number:

	$ bin/serverledge-cli -host 0.0.0.0 -port 1323 -cmd create -function func -memory 600 -src examples/hello.py -runtime python310 -handler "hello.handler" 

Invoke a function `func` with parameters `a=2` and `b=3`, you can optionally specify host and port number:

	$ bin/serverledge-cli -host 0.0.0.0 -port 1323 -cmd invoke -function func -param "a:2" -param "b:3"

You can optionally specify a QoS class name and a maximum requested response
time:

	$ bin/serverledge-cli invoke -function func -param ... -qosclass <class> -qosrespt <respt>

Get Server Status:
 
    $ bin/serverledge-cli status -host {host ip-address} -port {specific port}


To shutdown the etcd server:

	$ ./scripts/stop-etcd.sh

## Distributed Deployment

[This repository](https://github.com/grussorusso/serverledge-deploy) provides an Ansible playbook to deploy Serverledge in an 
Edge-Cloud environment.

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
An example is given in `examples/sieve.js`.

Create a function sieve:

    $ bin/serverledge-cli -host 0.0.0.0 -port 1323 -cmd create -function sieve -memory 128 -src examples/sieve.js -runtime nodejs17 -handler "sieve.js"

Invoke sieve function: 

    $ bin/serverledge-cli -host 0.0.0.0 -port 1323 -cmd invoke -function sieve 