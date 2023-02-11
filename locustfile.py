from locust import HttpUser, task, between

class TriggerMigration(HttpUser):
    wait_time = between(1, 1)

    @task
    def hello_world(self):
        self.client.post("/invoke/func",json={"Params":{"n":"5000"},"Async":True}
)
