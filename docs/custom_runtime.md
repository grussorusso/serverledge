# Custom image (the easy way)

The easiest way to build a custom function image is by leveraging the
Serverledge base runtime image, i.e., `grussorusso/serverledge-base`.
This image contains a simple implementation of the [Executor](https://github.com/grussorusso/serverledge/blob/main/docs/executor.md)
server. When the function is invoked, the Executor runs a user-specified
command as a new process and sets a few environment variables that may be
used by the called process.

- `PARAMS`: a JSON-encoded representation of the function parameters
- `RESULT_FILE`: name of the file where the function must write its JSON-encoded result
- `CONTEXT`: (optional) a JSON-encoded representation of the execution context

You can write a `Dockerfile` as follows to build your own runtime image, e.g.:

	FROM grussorusso/serverledge-base as BASE

	# Extend any image you want, e.g.;
	FROM tensorflow/tensorflow:latest

	# Required: install the executor as /executor
	COPY --from=BASE /executor /
	CMD /executor

	# Required: this is the command representing your function
	ENV CUSTOM_CMD "python /function.py"

	# Install your code and any dependency, e.g.:
	RUN pip3 install pillow
	COPY function.py /
	# ...

The new function must be created setting `custom` as the desired runtime and
specifying a `custom_image`.

	bin/serverledge-cli create -function myfunc -memory 256 -runtime custom -custom_image MY_IMAGE_TAG 

# Custom image (the harder way)

For higher efficiency, instead of using the default Executor implementation,
you can define your image from scratch including
your own Executor server, written in the language you prefer based on
the same protocol used by the Serverledge Executor.

By doing so, you may get rid of some process creation overheads, as
your function is directly called upon arrival of invocation requests.

