import os
import json

# Set by Executor
result_file = os.environ["RESULT_FILE"]
params_file = os.environ["PARAMS_FILE"]
params = {}
if params_file != "":
    with open(params_file, "rb") as fp:
        params = json.load(fp)
result = {}


with open(result_file, "w") as outf:
    result["Params"] = params
    result["Message"] = "Hello!"

    outf.write(json.dumps(result))

