## Serverledge API Reference


<!--

<details>
 <summary><code>POST</code> <code><b>/create</b></code> <code>(registers a new function)</code></summary>
 Details
</details>
-->


### Registering a new function

 <code>POST</code> <code><b>/create</b></code> <code>(registers a new function)</code>

##### Parameters

> | name      |  required   | type               | description                                                           |
> |-----------|-------------|-------------------------|------------|
> | `Name`    |         yes | string  | Name of the function (globally unique)  |
> | `Runtime`         | yes | string  | Base container runtime (e.g., `python310`)
> | `MemoryMB`        | yes | int     | Memory (in MB) reserved for each function instance
> | `CPUDemand`       |     | float   | Max CPU cores (or fractions of) allocated to function instances (e.g., `1.0` means up to 1 core, `-1.0` means no cap)
> | `Handler`         | (yes)    | string  | Function entrypoint in the source package; syntax and semantics depend on the chosen runtime (e.g., `module.function_name`). Not needed if `Runtime` is `custom`
> | `TarFunctionCode` | (yes)    | string  | Source code package as a base64-encoded TAR archive. Not needed if `Runtime` is `custom`
> | `CustomImage`     |     | string  | If `Runtime` is `custom`: custom container image to use


##### Responses

> | http code     | content-type                      | response                        | comments                                    |
> |---------------|-----------------------------------|---------------------------------|-----------------------------------|
> | `200`         | `application/json`        | `{ "Created": "function_name" }`    |                            |
> | `404`         | `text/plain`              | `Invalid runtime.` |    Chosen `Runtime` does not exist      |
> | `409`         | `text/plain`              |  |    Function already exists                        |
> | `503`         | `text/plain`              |  |    Creation failed                        |



------------------------------------------------------------------------------------------

#### Listing existing stubs & proxy configs as YAML string
