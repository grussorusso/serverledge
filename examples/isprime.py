def handler(params, context):
    try:
        n = int(params["n"])
        print(f"Checking n = {n}")
        result = is_prime(n)
        return {"IsPrime": result}
    except:
        return {}


def is_prime(n):
    for i in range(2, n//2):
        if n%i == 0:
            return False
    return True

