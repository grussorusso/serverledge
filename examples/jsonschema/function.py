import json
import jsonschema

schema = {
        "type" : "object",
        "properties" : {
            "age" : {"type" : "number"},
            "name" : {"type" : "string"},
            "company" : {"type" : "string"},
            },
        }

def handler (params, context):
    try:
        # validate json comprised in input 
        jsonschema.validate(instance=params, schema=schema)
        result = {"Validation": True}
    except Exception as e:
        print(e)
        result = {"Validation": False, "Error": str(e)}
    return result
