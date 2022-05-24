Functions are executed within containers. In the following, we will describe
how incoming requests are served within containers, assuming that a warm 
container for the function exists.

Each function container must run an **Executor** server, which listens for
HTTP requests on port `8080` (by default).

When a function request is scheduled for local execution within a warm container,
an invocation request is sent to the Executor as follows:

 - URL: `<container IP>:<executor port>/invoke`
 
 - Method: `POST`
 
 - Body: an `executor.InvocationRequest` (JSON-encoded)

 - Response (on success): an `executor.InvocationResult` (JSON-encoded)

An `InvocationRequest` has the following fields:

```
type InvocationRequest struct {
	Command    []string
	Params     map[string]interface{}
	Handler    string
	HandlerDir string
}
```

- `Command` (runtime-dependent; optional, depending on the Executor implementation): the
  command that the Executor has to run upon reception of a new request. E.g., 
  for a Python runtime, it may be set as `python /entrypoint.py`.

- `Params`: user-specified function parameters.

- `Handler` (runtime-dependent): identifier of the function to be executed. 
E.g., for Python runtimes, `<module_name>.<function_name>`.

- `HandlerDir`: directory where the function code has been copied.

The following object is returned upon function completion (or failure):

```
type InvocationResult struct {
	Success  bool
	Result   string
}
```

- `Success`: whether the function has been successfully executed.

- `Result`: what the function returned.


