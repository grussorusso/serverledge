## Running a C++ function

In this example, we show how a C++ function can be deployed and executed in
Serverledge. Compared to other languages (e.g., Python), more effort is required
as Serverledge does not currently provide built-in support for C++. Therefore,
we will need to build a custom container image for this purpose (more
information can be found [here](../../docs/custom_runtime.md)).
The image will contain (1) the Executor component of Serverledge and (2) the 
compiled C++ code of the function.

The file `function.cpp` implements a simple function that takes 2 integer
parameters `a` and `b`, and returns their sum.
Most the code in the file is actually devoted to parsing the JSON-serialized 
parameters for the function, and encoding the returned JSON object.

To build the image and register the function in Serverledge:

	docker build -t cpp-example .
	serverledge-cli create -function cpp -memory 256 -runtime custom -custom_image cpp-example 

To invoke the function:

	serverledge-cli invoke --function cpp --params_file example.json

where `example.json` contains the input parameters:

	{
		"a":3,
		"b":42
	}

