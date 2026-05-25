import os
import ssl
from pathlib import Path

import paho.mqtt.client as mqtt

# Configurare
BROKER = os.environ.get("MQTT_BROKER", "127.0.0.1")
PORT = int(os.environ.get("MQTT_PORT", "8883"))

PROJECT_ROOT = Path(__file__).resolve().parents[1]

# Căile către certificate
SECRETS_DIR = PROJECT_ROOT / "secrets"
CA_CRT = os.path.join(SECRETS_DIR, "ca.crt")
CLIENT_CRT = os.environ.get("MQTT_CLIENT_CRT", os.path.join(SECRETS_DIR, "web.crt"))
CLIENT_KEY = os.environ.get("MQTT_CLIENT_KEY", os.path.join(SECRETS_DIR, "web.key"))

# Crearea clientului MQTT
client = mqtt.Client(client_id="device-id")

# Configurarea TLS
client.tls_set(
    ca_certs=CA_CRT,
    certfile=CLIENT_CRT,
    keyfile=CLIENT_KEY,
    tls_version=ssl.PROTOCOL_TLSv1_2,
)
client.tls_insecure_set(os.environ.get("MQTT_INSECURE", "").lower() in {"1", "true", "yes"})

# Conectare
client.connect(BROKER, PORT, 60)
