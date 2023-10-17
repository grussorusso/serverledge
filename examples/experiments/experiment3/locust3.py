import json
import random
import time

import locust.stats
from locust import HttpUser, task, events, constant_throughput

locust.stats.PERCENTILES_TO_REPORT = [0.25, 0.50, 0.75, 0.80, 0.90, 0.95, 0.98, 0.99, 1.0]


class ResponseTimeLogger:

    def __init__(self, max_users: int):
        self.out_file = open(f"exp_3_resptimes_distribution_{max_users}_max_users.csv", "w")
        self.log(f"response_time,timestamp,duration")

    def log(self, rt):
        print(f"{rt}", file=self.out_file)

    def flush(self):
        self.out_file.flush()


logger: ResponseTimeLogger


@events.test_start.add_listener
def _(environment, **kw):
    global logger
    logger = ResponseTimeLogger(environment.runner.user_count)


class ServerledgeUser(HttpUser):
    # wait_time = between(0.1,0.2)
    wait_time = constant_throughput(5.0)

    @task
    def index(self):
        dice = random.random()
        if dice <= 0.2:
            qosClass = 1
        else:
            qosClass = 0
        self.client.post(f"/play/multifn_sequence", data=json.dumps({
            "Params": {"input": 1},
            "QoSClass": qosClass,
            "CanDoOffloading": True
        }), headers={'content-type': 'application/json'})

    def on_stop(self):
        logger.flush()


@events.request.add_listener
def composition_request_handler(request_type, name, response_time, response_length, response,
                                context, exception, start_time, url, **kwargs):
    if not exception:
        resp = json.loads(response.text)
        duration = resp["ResponseTime"]
        timestamp = time.time()
        logger.log(f"{response_time},{timestamp},{duration:.5f}")
    else:
        print(exception)
