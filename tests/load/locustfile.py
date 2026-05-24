from locust import HttpUser, task, between
import random

class MedSecAPIUser(HttpUser):
    wait_time = between(1, 5)
    token = None

    def on_start(self):
        # Authenticate on start
        response = self.client.post("/login", json={
            "email": "doctor@medsec.ro",
            "password": "doctor123"
        })
        if response.status_code == 200:
            self.token = response.json().get("token")
    @task(3)
    def view_photos(self):
        if self.token:
            self.client.get("/photos", headers={"Authorization": f"Bearer {self.token}"})

    @task(1)
    def view_devices(self):
        if self.token:
            self.client.get("/devices", headers={"Authorization": f"Bearer {self.token}"})

    @task(2)
    def view_dashboard(self):
        if self.token:
            self.client.get("/statistics", headers={"Authorization": f"Bearer {self.token}"})

    @task(1)
    def get_broker_info(self):
        self.client.get("/broker-info")
