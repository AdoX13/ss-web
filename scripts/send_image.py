#!/usr/bin/env python3
import argparse
import io
import json
import os
import socket
import ssl
import sys
import time

import paho.mqtt.client as mqtt
from PIL import Image, ImageDraw

# Configuration
BROKER = os.environ.get("MQTT_BROKER", "127.0.0.1")
PORT = int(os.environ.get("MQTT_PORT", "8883"))  # mTLS MQTT

# ===== CHANGE THIS FOR EACH DEVICE =====
DEVICE_ID = "python-sender-1"  # Unique ID for this device
DEVICE_NAME = "Python Test Device"  # Human-readable name
# ========================================

# Topics based on device ID
REGISTER_TOPIC = f"register/{DEVICE_ID}"
# Fixed topic to match server subscription (ssproject/images/#)
PHOTO_TOPIC = f"ssproject/images/{DEVICE_ID}"

# Get absolute paths
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.dirname(SCRIPT_DIR)

SECRETS_DIR = os.path.join(PROJECT_ROOT, "secrets")
CA_CRT = os.path.join(SECRETS_DIR, "ca.crt")
CLIENT_CRT = os.environ.get("MQTT_CLIENT_CRT", os.path.join(SECRETS_DIR, "web.crt"))
CLIENT_KEY = os.environ.get("MQTT_CLIENT_KEY", os.path.join(SECRETS_DIR, "web.key"))


def get_local_ip():
    """Get the local IP address of this machine"""
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(("8.8.8.8", 80))
        ip = s.getsockname()[0]
        s.close()
        return ip
    except:
        return "unknown"


def create_test_image(device_id):
    """Create a test image with timestamp and device ID"""
    img = Image.new("RGB", (300, 150), color="white")
    d = ImageDraw.Draw(img)
    d.text((10, 20), f"Device: {device_id}", fill="black")
    d.text((10, 50), f"Time: {time.strftime('%H:%M:%S')}", fill="black")
    d.text((10, 80), "HELLO ADMIN", fill="blue")

    img_byte_arr = io.BytesIO()
    img.save(img_byte_arr, format="JPEG")
    return img_byte_arr.getvalue()


def load_image_from_file(path):
    """Load image from file and convert to bytes"""
    try:
        with Image.open(path) as img:
            if img.mode in ("RGBA", "P"):
                img = img.convert("RGB")

            img_byte_arr = io.BytesIO()
            img.save(img_byte_arr, format="JPEG")
            return img_byte_arr.getvalue()
    except Exception as e:
        print(f"Error loading image {path}: {e}")
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(description="Send an image via MQTT")
    parser.add_argument("file", nargs="?", help="Path to image file (optional)")
    parser.add_argument("--broker", default="127.0.0.1", help="Broker host")
    parser.add_argument("--port", type=int, default=8883, help="Broker port (8883 for mTLS, 1883 for plaintext)")
    parser.add_argument("--device-id", default="python-sender-1", help="Device ID")
    parser.add_argument("--device-name", default="Python Test Device", help="Device Name")
    parser.add_argument("--no-tls", action="store_true", help="Disable mTLS (use plaintext)")
    parser.add_argument("--ca", default=DEFAULT_CA_CRT, help="CA Cert path")
    parser.add_argument("--cert", default=DEFAULT_CLIENT_CRT, help="Client Cert path")
    parser.add_argument("--key", default=DEFAULT_CLIENT_KEY, help="Client Key path")
    parser.add_argument("--verbose", action="store_true", help="Enable verbose logging")
    
    args = parser.parse_args()

    register_topic = f"register/{args.device_id}"
    photo_topic = f"ssproject/images/{args.device_id}"

    if not args.no_tls:
        if args.port == 1883:
            print("Warning: Using port 1883 but mTLS is enabled. Usually mTLS uses 8883.")
        for path_name, path in [("CA", args.ca), ("Cert", args.cert), ("Key", args.key)]:
            if not os.path.exists(path):
                print(f"Error: {path_name} file not found: {path}")
                print("Run ./scripts/gen_certs.sh to generate certificates, or use --no-tls")
                sys.exit(1)

    # Callbacks
    def on_connect(client, userdata, flags, rc):
        if rc == 0:
            if args.verbose:
                print(f"Connected to MQTT Broker at {args.broker}:{args.port}")
            
            local_ip = get_local_ip()
            registration = json.dumps({
                "name": args.device_name, 
                "ip": local_ip, 
                "port": str(args.port)
            })
            if args.verbose:
                print(f"Registering device: {register_topic}")
            client.publish(register_topic, registration)
            time.sleep(0.5)

            if args.verbose:
                print(f"Publishing image to: {photo_topic}")

            if args.file:
                if os.path.isdir(args.file):
                    print(f"Error: {args.file} is a directory. Please specify an image file.")
                    sys.exit(1)
                image_data = load_image_from_file(args.file)
                if args.verbose:
                    print(f"Sending file: {args.file}")
            else:
                image_data = create_test_image(args.device_id)
                if args.verbose:
                    print("Sending generated test image")

            client.publish(photo_topic, image_data)
        else:
            print(f"Failed to connect, return code {rc}")
            sys.exit(1)

    def on_publish(client, userdata, mid):
        if mid == 2:
            print(f"✅ Device '{args.device_id}' registered and photo sent!")
            if args.verbose:
                print(f"   Topic: {photo_topic}")
            client.disconnect()
            sys.exit(0)

    client = mqtt.Client(client_id=args.device_id)
    client.on_connect = on_connect
    client.on_publish = on_publish

    if not args.no_tls:
        client.tls_set(
            ca_certs=args.ca,
            certfile=args.cert,
            keyfile=args.key,
            tls_version=ssl.PROTOCOL_TLSv1_2,
        )
        client.tls_insecure_set(True)

    if args.verbose:
        print(f"Device ID: {args.device_id}")
        print(f"Connecting to {args.broker}:{args.port}...")
    
    try:
        client.connect(args.broker, args.port, 60)
        client.loop_forever()
    except Exception as e:
        print(f"Connection failed: {e}")
        sys.exit(1)


def on_publish(client, userdata, mid):
    # Disconnect after second publish (the photo)
    # Note: connect sends no message, register is mid=1, photo is mid=2
    if mid == 2:
        print("Message published successfully!")
        print(f"\n✅ Device '{DEVICE_ID}' registered and photo sent!")
        print(f"   Topic: {PHOTO_TOPIC}")
        client.disconnect()
        sys.exit(0)


# Create MQTT client
client = mqtt.Client(client_id=DEVICE_ID)
client.on_connect = on_connect
client.on_publish = on_publish

client.tls_set(
    ca_certs=CA_CRT,
    certfile=CLIENT_CRT,
    keyfile=CLIENT_KEY,
    tls_version=ssl.PROTOCOL_TLSv1_2,
)
client.tls_insecure_set(os.environ.get("MQTT_INSECURE", "").lower() in {"1", "true", "yes"})

print(f"Device ID: {DEVICE_ID}")
print(f"Connecting to {BROKER}:{PORT}...")
try:
    client.connect(BROKER, PORT, 60)
    client.loop_forever()
except Exception as e:
    print(f"Connection failed: {e}")
    sys.exit(1)
