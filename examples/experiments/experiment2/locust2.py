import json
import random
import time

import locust.stats
from locust import HttpUser, task, events, constant_throughput

locust.stats.PERCENTILES_TO_REPORT = [0.25, 0.50, 0.75, 0.80, 0.90, 0.95, 0.98, 0.99, 1.0]


class ResponseTimeLogger:

    def __init__(self):
        self.out_file = open(f"exp_2_resptimes_complex.csv", "w")
        self.log(f"response_time,timestamp,duration,task")

    def log(self, rt):
        print(f"{rt}", file=self.out_file)

    def flush(self):
        self.out_file.flush()


logger: ResponseTimeLogger = ResponseTimeLogger()
i: int = 0


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
        global i
        if i % 3 == 0:  # word_count
            self.client.post(f"/play/complex", data=json.dumps({
                "Params": {
                    "InputText": "Word counting is a useful technique for analyzing text data. It helps in various "
                                 "natural language processing tasks. In this example, we are testing the wordCount "
                                 "function in JavaScript. It should accurately count the number of words in this "
                                 "text. Counting words can be a fundamental step in text analysis.",
                    "Task": True,
                },
                "QoSClass": qosClass,
                "CanDoOffloading": True
            }), headers={'content-type': 'application/json'})
        elif i % 3 == 1:  # summarize
            self.client.post(f"/play/complex", data=json.dumps({
                "Params": {
                    "InputText": "The Solar System consists of the Sun and all the celestial objects bound to it by "
                                 "gravity, including the eight major planets and their moons, asteroids, comets, "
                                 "and more. The Sun is a star located at the center of the Solar System. It provides "
                                 "light, heat, and energy, making life possible on Earth.\n\nThe eight major planets "
                                 "in our Solar System are Mercury, Venus, Earth, Mars, Jupiter, Saturn, Uranus, "
                                 "and Neptune. Each planet has unique characteristics, and some have moons of their "
                                 "own. For example, Earth has one natural satellite, the Moon.\n\nAsteroids are rocky "
                                 "objects that orbit the Sun, mainly found in the asteroid belt between the orbits of "
                                 "Mars and Jupiter. Comets are icy bodies that develop tails when they approach the "
                                 "Sun.\n\nStudying the Solar System provides insights into the formation and "
                                 "evolution of celestial bodies, as well as the potential for extraterrestrial life. "
                                 "Scientists use various tools and telescopes to explore and learn more about the "
                                 "mysteries of our Solar System.\n",
                    "Task": False
                },
                "QoSClass": qosClass,
                "CanDoOffloading": True
            }), headers={'content-type': 'application/json'})
        else:  # parallel grep
            self.client.post(f"/play/complex", data=json.dumps({
                "Params": {
                    "InputText": [
                        "This is an example text for testing the grep function.\nYou can use the grep function to "
                        "search for specific words or patterns in text.\nThe function is a powerful tool for text "
                        "processing.\n",
                        "It allows you to filter and extract lines that match a given pattern.\nYou can customize the "
                        "pattern using regular expressions.\nFeel free to test the grep function with different "
                        "patterns and texts."
                    ]
                },
                "QoSClass": qosClass,
                "CanDoOffloading": True
            }), headers={'content-type': 'application/json'})

        i += 1

    def on_stop(self):
        logger.flush()


@events.request.add_listener
def composition_request_handler(request_type, name, response_time, response_length, response,
                                context, exception, start_time, url, **kwargs):
    if not exception:
        resp = json.loads(response.text)
        duration = resp["ResponseTime"]
        timestamp = time.time()
        global i
        fn: str
        if i % 3 == 0:
            fn = "word_count"
        elif i % 3 == 1:
            fn = "summarize"
        else:
            fn = "grep"

        logger.log(f"{response_time},{timestamp},{duration:.5f},{fn}")
    else:
        print(exception)
