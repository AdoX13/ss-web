#!/usr/bin/env python3
import time
import random
import os
import threading
from concurrent.futures import ThreadPoolExecutor

# Use the bulk_send script logic to simulate load
import sys
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
sys.path.append(os.path.join(os.path.dirname(SCRIPT_DIR), '..', 'scripts'))

try:
    from bulk_send import send_image
except ImportError:
    print("Could not import bulk_send. Ensure scripts/bulk_send.py exists.")
    sys.exit(1)

class DummyArgs:
    def __init__(self, device_id):
        self.device_id = device_id
        self.broker = "127.0.0.1"
        self.port = 8883
        self.no_tls = False
        
        project_root = os.path.join(os.path.dirname(SCRIPT_DIR), '..')
        self.ca = os.path.join(project_root, "secrets", "ca.crt")
        self.cert = os.path.join(project_root, "secrets", "python-sender.crt")
        self.key = os.path.join(project_root, "secrets", "python-sender.key")

def worker(worker_id, image_path, num_messages):
    args = DummyArgs(f"load-test-device-{worker_id}")
    success = 0
    
    for i in range(num_messages):
        res = send_image(image_path, args, i)
        if res is True:
            success += 1
        time.sleep(random.uniform(0.1, 0.5))
        
    return success

def main():
    project_root = os.path.join(os.path.dirname(SCRIPT_DIR), '..')
    image_dir = os.path.join(project_root, "medical-images", "synthetic")
    
    if not os.path.exists(image_dir):
        print(f"Error: {image_dir} not found. Run generate_synthetic_certificates.py first.")
        return

    images = [os.path.join(image_dir, f) for f in os.listdir(image_dir) if f.endswith('.jpg')]
    if not images:
        print("No synthetic images found.")
        return

    num_workers = 10
    msgs_per_worker = 20
    
    print(f"Starting MQTT Load Test: {num_workers} devices sending {msgs_per_worker} images each over mTLS...")
    start_time = time.time()
    
    total_success = 0
    with ThreadPoolExecutor(max_workers=num_workers) as executor:
        futures = []
        for i in range(num_workers):
            img = random.choice(images)
            futures.append(executor.submit(worker, i, img, msgs_per_worker))
            
        for future in futures:
            total_success += future.result()
            
    duration = time.time() - start_time
    print(f"Test Complete in {duration:.2f}s")
    print(f"Successfully sent {total_success} / {num_workers * msgs_per_worker} images.")
    print(f"Throughput: {total_success / duration:.2f} images/sec")

if __name__ == "__main__":
    main()
