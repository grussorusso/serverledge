from locust import HttpUser, task, between

class SwarmServerLedge(HttpUser):
    wait_time = between(5, 10) # 3 - 5 s
    
    @task
    def userInvoke(self):
        self.client.post("/invoke/func",json={"Params":{"n":"100000"},"Async":False}
)
