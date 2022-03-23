# Custom image (the easy way)

The easiest way to build a custom function image is by extending one of the
Serverledge runtime images, e.g., `grussorusso/serverledge-python310`.

You can write a `Dockerfile` to extend the image as desired:

	FROM grussorusso/serverledge-python310

	# Copy/install what you need
	COPY myfile.py /
	
	# This is the command representing your own function
	ENV CUSTOM_CMD "python /myfile.py" 

	CMD /executor   # Do not change this line


The new function must be created setting `custom` as the desired runtime and
specifying a `custom_image`.

	bin/serverledge-cli create -function myfunc -memory 256 -runtime custom -custom_image MY_IMAGE_TAG 

# Custom image (a harder way)

A slightly different approach allows you to build the custom image without 
building on top of one of the available runtime images.
In this case, you need to compile and install the Serverledge Executor
in `/executor`, setting it as the container `CMD`.
As shown above, you also need to set the `CUSTOM_CMD` environment variable 
specifying the command to be executed by the Executor upon invocation.

# Custom image (harder way)

For higher efficiency, you can define your image from scratch implementing
your own Executor server, which must run within the container and implement
the same protocol used by the Serverledge Executor component for invocation.

By doing so, you may get rid of some process creation overheads.

