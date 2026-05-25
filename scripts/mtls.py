#!/usr/bin/env python3
"""
mTLS connectivity test for the MQTT broker.

Connects to the Mosquitto broker using mTLS client certificates
and verifies the TLS handshake succeeds. Used for Lab 7 verification.

Usage:
    python3 scripts/mtls.py                          # Default: localhost:8883
    python3 scripts/mtls.py --broker 192.168.1.95    # Custom broker host
    python3 scripts/mtls.py --port 8883              # Custom port
"""

import argparse
import os
import ssl
import sys
import time

import paho.mqtt.client as mqtt

# Resolve paths relative to this script (not CWD)
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.dirname(SCRIPT_DIR)
SECRETS_DIR = os.path.join(PROJECT_ROOT, "secrets")

# Default certificate paths
CA_CRT = os.path.join(SECRETS_DIR, "ca.crt")
CLIENT_CRT = os.environ.get("MQTT_CLIENT_CRT", os.path.join(SECRETS_DIR, "web.crt"))
CLIENT_KEY = os.environ.get("MQTT_CLIENT_KEY", os.path.join(SECRETS_DIR, "web.key"))


def on_connect(client, userdata, flags, rc):
    if rc == 0:
        print("✅ mTLS connection successful!")
        print(f"   Broker: {userdata['broker']}:{userdata['port']}")
        print(f"   CA:     {userdata['ca']}")
        print(f"   Cert:   {userdata['cert']}")
        print(f"   Key:    {userdata['key']}")

        # Subscribe to a system topic to verify we can communicate
        client.subscribe("$SYS/broker/uptime", qos=0)
    else:
        rc_messages = {
            1: "Incorrect protocol version",
            2: "Invalid client identifier",
            3: "Server unavailable",
            4: "Bad username or password",
            5: "Not authorized",
        }
        msg = rc_messages.get(rc, f"Unknown error (code {rc})")
        print(f"❌ Connection failed: {msg}")
        sys.exit(1)


def on_message(client, userdata, msg):
    print(f"   Received on {msg.topic}: {msg.payload.decode()}")
    print("\n✅ Full mTLS round-trip verified!")
    client.disconnect()


def on_disconnect(client, userdata, rc):
    if rc == 0:
        print("   Disconnected cleanly.")
    sys.exit(0)


def main():
    parser = argparse.ArgumentParser(
        description="Test mTLS connectivity to the MQTT broker (Lab 7)"
    )
    parser.add_argument(
        "--broker", default="127.0.0.1", help="Broker hostname (default: 127.0.0.1)"
    )
    parser.add_argument(
        "--port", type=int, default=8883, help="Broker port (default: 8883)"
    )
    parser.add_argument(
        "--ca", default=CA_CRT, help=f"CA certificate path (default: {CA_CRT})"
    )
    parser.add_argument(
        "--cert",
        default=CLIENT_CRT,
        help=f"Client certificate path (default: {CLIENT_CRT})",
    )
    parser.add_argument(
        "--key",
        default=CLIENT_KEY,
        help=f"Client key path (default: {CLIENT_KEY})",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=10,
        help="Connection timeout in seconds (default: 10)",
    )
    args = parser.parse_args()

    # Validate cert files exist
    for label, path in [("CA", args.ca), ("Cert", args.cert), ("Key", args.key)]:
        if not os.path.isfile(path):
            print(f"❌ {label} file not found: {path}")
            print("   Run: ./scripts/gen_certs.sh")
            sys.exit(1)

    print(f"🔐 Testing mTLS connection to {args.broker}:{args.port}")
    print(f"   CA:   {args.ca}")
    print(f"   Cert: {args.cert}")
    print(f"   Key:  {args.key}")
    print()

    # Store connection info for callbacks
    userdata = {
        "broker": args.broker,
        "port": args.port,
        "ca": args.ca,
        "cert": args.cert,
        "key": args.key,
    }

    client = mqtt.Client(client_id="mtls-test", userdata=userdata)
    client.on_connect = on_connect
    client.on_message = on_message
    client.on_disconnect = on_disconnect

    # Configure TLS with mTLS client certificates
    client.tls_set(
        ca_certs=args.ca,
        certfile=args.cert,
        keyfile=args.key,
        tls_version=ssl.PROTOCOL_TLSv1_2,
    )
    # Skip hostname verification for local development
    client.tls_insecure_set(True)

    try:
        client.connect(args.broker, args.port, args.timeout)
        client.loop_start()
        time.sleep(args.timeout)
        print("⚠  Timeout — no response from broker within {args.timeout}s")
        client.disconnect()
        sys.exit(1)
    except ConnectionRefusedError:
        print(f"❌ Connection refused at {args.broker}:{args.port}")
        print("   Is the broker running? Try: docker compose up -d broker")
        sys.exit(1)
    except ssl.SSLError as e:
        print(f"❌ TLS handshake failed: {e}")
        print("   Check that certificates are valid and signed by the same CA.")
        sys.exit(1)
    except Exception as e:
        print(f"❌ Connection failed: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
