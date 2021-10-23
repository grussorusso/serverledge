import re
import os
import json
import sys

handler = os.environ["HANDLER"] 
handler_dir = os.environ["HANDLER_DIR"]
result_file = os.environ["RESULT_FILE"]

if "params" in os.environ:
    params = json.loads(os.environ["PARAMS"]) 
else:
    params = {}

if "context" in os.environ:
    context = json.loads(os.environ["CONTEXT"]) 
else:
    context = {}

sys.path.insert(1, handler_dir)

# Get module name
module,func_name = os.path.splitext(handler)
func_name = func_name[1:] # strip initial dot

# Import module
exec(f"import {module}")

# Call function
exec(f"result = {module}.{func_name}(params, context)")

#print(result)
with open(result_file, "w") as of:
    json.dump(result, of)

