#!/usr/bin/env python3
"""
Script pentru incarcarea automata a tuturor imaginilor dintr-un folder prin MQTT.
Scanează un folder și trimite toate imaginile (PNG, JPG, JPEG) la serverul MQTT.

Utilizare:
    python3 upload_folder.py /path/to/folder
    python3 upload_folder.py /path/to/folder --device-id medical-scanner-1
"""

import argparse
import json
import os
import socket
import ssl
import sys
import time
from pathlib import Path

import paho.mqtt.client as mqtt

# Configuration
BROKER = os.environ.get("MQTT_BROKER", "127.0.0.1")
PORT = int(os.environ.get("MQTT_PORT", "8883"))  # mTLS MQTT

# Default device info
DEFAULT_DEVICE_ID = "folder-uploader"
DEFAULT_DEVICE_NAME = "Folder Batch Upload"

# Supported image extensions
SUPPORTED_EXTENSIONS = {".png", ".jpg", ".jpeg", ".PNG", ".JPG", ".JPEG"}

# Get absolute paths
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.dirname(SCRIPT_DIR)
SECRETS_DIR = os.path.join(PROJECT_ROOT, "secrets")
CA_CRT = os.path.join(SECRETS_DIR, "ca.crt")
CLIENT_CRT = os.environ.get("MQTT_CLIENT_CRT", os.path.join(SECRETS_DIR, "web.crt"))
CLIENT_KEY = os.environ.get("MQTT_CLIENT_KEY", os.path.join(SECRETS_DIR, "web.key"))


class ImageUploader:
    def __init__(self, device_id, device_name):
        self.device_id = device_id
        self.device_name = device_name
        self.register_topic = f"register/{device_id}"
        self.photo_topic = f"ssproject/images/{device_id}"
        self.images_to_send = []
        self.current_index = 0
        self.connected = False
        self.registered = False

    def get_local_ip(self):
        """Get the local IP address of this machine"""
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            s.connect(("8.8.8.8", 80))
            ip = s.getsockname()[0]
            s.close()
            return ip
        except:
            return "unknown"

    def on_connect(self, client, userdata, flags, rc):
        if rc == 0:
            print(f"✓ Conectat la MQTT Broker ({BROKER}:{PORT})")
            self.connected = True

            # Register the device
            local_ip = self.get_local_ip()
            registration = json.dumps(
                {"name": self.device_name, "ip": local_ip, "port": str(PORT)}
            )
            print(f"✓ Înregistrare dispozitiv: {self.device_id}")
            client.publish(self.register_topic, registration)
            time.sleep(0.5)
            self.registered = True

            # Start sending images
            self.send_next_image(client)
        else:
            print(f"✗ Eroare conexiune, cod: {rc}")
            sys.exit(1)

    def on_publish(self, client, userdata, mid):
        # Send next image after current one is published
        time.sleep(0.3)  # Small delay between images
        self.send_next_image(client)

    def send_next_image(self, client):
        if self.current_index >= len(self.images_to_send):
            print(
                f"\n✓ Toate imaginile au fost trimise ({len(self.images_to_send)} imagini)"
            )
            client.disconnect()
            return

        image_path = self.images_to_send[self.current_index]
        try:
            with open(image_path, "rb") as f:
                image_data = f.read()

            client.publish(self.photo_topic, image_data)
            print(
                f"  [{self.current_index + 1}/{len(self.images_to_send)}] Trimis: {os.path.basename(image_path)}"
            )
            self.current_index += 1

        except Exception as e:
            print(f"✗ Eroare la citirea imaginii {image_path}: {e}")
            self.current_index += 1
            self.send_next_image(client)

    def upload_folder(self, folder_path):
        # Find all images in folder
        folder = Path(folder_path)
        if not folder.exists():
            print(f"✗ Folder-ul nu există: {folder_path}")
            return

        if not folder.is_dir():
            print(f"✗ Calea specificată nu este un folder: {folder_path}")
            return

        # Collect all image files
        for file_path in folder.iterdir():
            if file_path.is_file() and file_path.suffix in SUPPORTED_EXTENSIONS:
                self.images_to_send.append(str(file_path))

        if not self.images_to_send:
            print(f"✗ Nu s-au găsit imagini în folder-ul: {folder_path}")
            print(f"  Formate suportate: {', '.join(SUPPORTED_EXTENSIONS)}")
            return

        # Sort images by name for consistent ordering
        self.images_to_send.sort()

        print(f"\n📁 Folder: {folder_path}")
        print(f"📸 Găsite {len(self.images_to_send)} imagini")
        print(f"🔧 Device ID: {self.device_id}")
        print(f"📡 Server: {BROKER}:{PORT}\n")
        print("Se încarcă imaginile...\n")

        # Create MQTT client
        client = mqtt.Client(client_id=self.device_id)
        client.on_connect = self.on_connect
        client.on_publish = self.on_publish

        client.tls_set(
            ca_certs=CA_CRT,
            certfile=CLIENT_CRT,
            keyfile=CLIENT_KEY,
            tls_version=ssl.PROTOCOL_TLSv1_2,
        )
        client.tls_insecure_set(os.environ.get("MQTT_INSECURE", "").lower() in {"1", "true", "yes"})

        try:
            client.connect(BROKER, PORT, 60)
            client.loop_forever()
        except KeyboardInterrupt:
            print("\n\n⚠ Întrerupt de utilizator")
            client.disconnect()
        except Exception as e:
            print(f"\n✗ Eroare de conexiune: {e}")
            sys.exit(1)


def main():
    parser = argparse.ArgumentParser(
        description="Încarcă automat toate imaginile dintr-un folder prin MQTT",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Exemple de utilizare:
  %(prog)s                                    # Folosește folder-ul implicit: medical-images/
  %(prog)s /path/to/images                    # Specifică un folder custom
  %(prog)s --device-id medical-scanner-1      # Folder implicit cu device ID personalizat
  %(prog)s ~/Documents/fise_medicale --device-id scanner-cabinet
        """,
    )

    # Calculate default folder path
    default_folder = os.path.join(PROJECT_ROOT, "medical-images")

    parser.add_argument(
        "folder",
        nargs="?",  # Make optional
        default=default_folder,
        help=f"Calea către folder-ul cu imagini (default: {default_folder})",
    )
    parser.add_argument(
        "--device-id",
        default=DEFAULT_DEVICE_ID,
        help=f"ID-ul dispozitivului (default: {DEFAULT_DEVICE_ID})",
    )
    parser.add_argument(
        "--device-name",
        default=DEFAULT_DEVICE_NAME,
        help=f"Numele dispozitivului (default: {DEFAULT_DEVICE_NAME})",
    )

    args = parser.parse_args()

    uploader = ImageUploader(args.device_id, args.device_name)
    uploader.upload_folder(args.folder)


if __name__ == "__main__":
    main()
