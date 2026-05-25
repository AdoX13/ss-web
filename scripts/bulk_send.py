#!/usr/bin/env python3
import argparse
import json
import os
import ssl
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed

import paho.mqtt.client as mqtt

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.dirname(SCRIPT_DIR)
SECRETS_DIR = os.path.join(PROJECT_ROOT, "secrets")

DEFAULT_CA_CRT = os.path.join(SECRETS_DIR, "ca.crt")
DEFAULT_CLIENT_CRT = os.path.join(SECRETS_DIR, "python-sender.crt")
DEFAULT_CLIENT_KEY = os.path.join(SECRETS_DIR, "python-sender.key")
DEFAULT_FOLDER = os.path.join(PROJECT_ROOT, "medical-images", "synthetic")

def send_image(image_path, args, index):
    client = mqtt.Client(client_id=f"{args.device_id}-{index}")
    
    if not args.no_tls:
        client.tls_set(
            ca_certs=args.ca,
            certfile=args.cert,
            keyfile=args.key,
            tls_version=ssl.PROTOCOL_TLSv1_2,
        )
        client.tls_insecure_set(True)

    try:
        client.connect(args.broker, args.port, 10)
    except Exception as e:
        return f"Failed to connect: {e}"

    # Wait for connect briefly
    client.loop_start()
    time.sleep(0.5)

    with open(image_path, "rb") as f:
        image_data = f.read()

    photo_topic = f"ssproject/images/{args.device_id}"
    msg_info = client.publish(photo_topic, image_data, qos=1)
    msg_info.wait_for_publish(timeout=5)
    
    client.disconnect()
    client.loop_stop()

    if msg_info.is_published():
        return True
    return "Failed to publish"

def main():
    parser = argparse.ArgumentParser(description="Bulk send images via MQTT concurrently")
    parser.add_argument("--folder", default=DEFAULT_FOLDER, help="Directory containing images")
    parser.add_argument("--broker", default="127.0.0.1", help="Broker host")
    parser.add_argument("--port", type=int, default=8883, help="Broker port")
    parser.add_argument("--device-id", default="bulk-sender-1", help="Device ID")
    parser.add_argument("--concurrency", type=int, default=5, help="Number of concurrent uploads")
    parser.add_argument("--no-tls", action="store_true", help="Disable mTLS (use plaintext)")
    parser.add_argument("--ca", default=DEFAULT_CA_CRT, help="CA Cert path")
    parser.add_argument("--cert", default=DEFAULT_CLIENT_CRT, help="Client Cert path")
    parser.add_argument("--key", default=DEFAULT_CLIENT_KEY, help="Client Key path")
    
    args = parser.parse_args()

    if not os.path.exists(args.folder):
        print(f"Error: Folder {args.folder} does not exist.")
        sys.exit(1)

    images = [os.path.join(args.folder, f) for f in os.listdir(args.folder) if f.lower().endswith(('.png', '.jpg', '.jpeg'))]
    
    if not images:
        print(f"No images found in {args.folder}")
        sys.exit(1)

    print(f"Found {len(images)} images. Uploading with concurrency {args.concurrency}...")

    # First, register the device once
    client = mqtt.Client(client_id=args.device_id)
    if not args.no_tls:
        client.tls_set(ca_certs=args.ca, certfile=args.cert, keyfile=args.key, tls_version=ssl.PROTOCOL_TLSv1_2)
        client.tls_insecure_set(True)
    
    try:
        client.connect(args.broker, args.port, 10)
        client.loop_start()
        registration = json.dumps({"name": "Bulk Sender", "ip": "127.0.0.1", "port": str(args.port)})
        client.publish(f"register/{args.device_id}", registration, qos=1).wait_for_publish(2)
        client.disconnect()
        client.loop_stop()
    except Exception as e:
        print(f"Initial registration failed: {e}")
        sys.exit(1)

    success_count = 0
    failure_count = 0

    with ThreadPoolExecutor(max_workers=args.concurrency) as executor:
        futures = {executor.submit(send_image, img, args, i): img for i, img in enumerate(images)}
        
        for future in as_completed(futures):
            img = futures[future]
            try:
                result = future.result()
                if result is True:
                    success_count += 1
                    print(f"✅ Sent {os.path.basename(img)}")
                else:
                    failure_count += 1
                    print(f"❌ Failed {os.path.basename(img)}: {result}")
            except Exception as e:
                failure_count += 1
                print(f"❌ Error {os.path.basename(img)}: {e}")

    print(f"\nUpload complete: {success_count} succeeded, {failure_count} failed.")

if __name__ == "__main__":
    main()
