from locust import HttpUser, task, between

class ApiUser(HttpUser):
    wait_time = between(0,0)

    @task(3)
    def ping(self):
        self.client.get("/ping/")

    @task(1)
    def cpu(self):
        self.client.get("/cpu/")

    @task(1)
    def io(self):
        self.client.get("/io/")