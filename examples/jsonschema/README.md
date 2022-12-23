This examples demonstrates how to define a function through a custom
container image. In particular, we define a Python function that validates
user-given JSON input according to a fixed schema. For this purpose, we use
the `jsonschema` Python library, which is not available in the default Python
runtime image.

The actual function code is in `function.py`. We also need to copy an Executor
implementation (see the docs) to the container. The file `executor.py` contains
an adapted version of the Executor implementation taken from the default Python
runtime image of Serverledge.

## Building the image

	$ docker build -t <IMAGETAG> .

## Using the image

	$ serverledge-cli create -f jsonFunc --memory 256 --runtime custom\
	    --custom_image <IMAGETAG>
	$ serverledge-cli invoke -f jsonFunc --params_file input.json
