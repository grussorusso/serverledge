import time

def increment(params, context):
    num = int(params["num"])
    num = num + 1
    time.sleep(1)
    return "The result is " + str(num)