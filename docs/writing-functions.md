# Writing functions

## Python

Available runtime: `python310` (Python 3.10)

	def handler_fun (context, params):
		return "..."

Specify the handler as `<module_name>.<function_name>` (e.g., `myfile.handler_fun`).
An example is given in `examples/hello.py`.

## NodeJS

Available runtime: `nodejs17` (NodeJS 17)

	function handler_fun (context, params) {
		return "..."
	}

	module.exports = handler_fun // this is mandatory!

Specify the handler as `<script_file_name>.js` (e.g., `myfile.js`).
An example is given in `examples/sieve.js`.

## Custom function runtimes

Follow [these instructions](./custom_runtime.md).
