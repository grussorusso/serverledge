import pickle

from locust import HttpUser, task, between


class ServerEdgeUser(HttpUser):
    wait_time = between(1, 2)

    @task(0)
    def create_python_function(self):
        # don't use this function, create func2 manually
        self.client.post("/create", json={"Name": "func2",
                                          "Handler": "hello.handler",
                                          "Runtime": "python310",
                                          "MemoryMB": 128,
                                          "CPUDemand": 1,
                                          "TarFunctionCode": "examples/hello.py"})
        return

    @task(3)
    def invoke_low_service_function(self):
        params = {"a": "1", "b": "2"}
        p = pickle.dumps(params)
        self.client.post("/invoke/func2", json={
            "Params": pickle.loads(p),
            "QoSClass": 0,
            "QoSMaxRespT": 3,
            "Offloading": True
        })

    @task(1)
    def invoke_high_perf_function(self):
        params = {"a": "1", "b": "2"}
        p = pickle.dumps(params)
        self.client.post("/invoke/func2", json={
            "Params": pickle.loads(p),
            "QoSClass": 1,
            "QoSMaxRespT": 1,
            "Offloading": True
        })

    @task(2)
    def invoke_high_availability_function(self):
        params = {"a": "1", "b": "2"}
        p = pickle.dumps(params)
        self.client.post("/invoke/func2", json={
            "Params": pickle.loads(p),
            "QoSClass": 2,
            "QoSMaxRespT": 5,
            "Offloading": True
        })

    def on_start(self):
        self.create_python_function()

#usage: locust -f load-test.py