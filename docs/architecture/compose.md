# Docker Compose Documentation

## Overview
The `docker-compose.yml` file is the orchestrator for the backend stack. It defines three core services, the network topology, persistent volumes, and secret injection.

## Services

### 1. `go-api`
- **Purpose**: The main REST API and MQTT message consumer. Runs the OCR pipeline and serves data to the React frontend.
- **Image**: Built from `./server` (Dockerfile).
- **Ports**: Exposes `8080` to the host.
- **Volumes**: Mounts `./uploads` to `/app/uploads` to store extracted photos locally.
- **Secrets**: Mounts `ca.crt`, `web.crt`, and `web.key` to allow mTLS connections to the broker.
- **Healthcheck**: Pings the `/broker-info` HTTP endpoint.

### 2. `broker`
- **Purpose**: Eclipse Mosquitto MQTT broker. Handles incoming image streams from IoT devices and ingestion scripts.
- **Image**: `eclipse-mosquitto:latest`
- **Ports**: Exposes `8883` for mTLS. Port `1883` is deliberately omitted.
- **Volumes**: Mounts `./broker/mosquitto.conf` as read-only.
- **Secrets**: Mounts `ca.crt`, `server.crt`, and `server.key` for TLS termination and client verification.
- **Healthcheck**: Uses `mosquitto_sub` to subscribe to `$SYS/broker/uptime` using mTLS credentials.

### 3. `mongo-db`
- **Purpose**: Persistent data store for users, devices, and medical records.
- **Image**: `mongo:latest`
- **Ports**: Maps container port `27017` to host port `27019`.
- **Command**: `--auth` forces authentication.
- **Volumes**: Uses the named volume `mongo-data` mapped to `/data/db`.
- **Healthcheck**: Uses `mongosh` to ping the admin database.

## Secrets Management
Docker secrets are defined at the bottom of the compose file, linking physical files in `./secrets/` to named secrets:

```yaml
secrets:
  ca.crt:
    file: ./secrets/ca.crt
```

When a service requests a secret (e.g., `- ca.crt`), Docker mounts it at `/run/secrets/ca.crt` inside the container. This is more secure than environment variables for large keys and certificates.

## Extensibility: Observability
To add monitoring, a separate override file (`docker-compose.observability.yml`) can be used. It defines Prometheus and Grafana services attached to the same `backend` network, allowing them to scrape metrics from the Go API.

Run it alongside the main stack:
```bash
docker compose -f docker-compose.yml -f docker-compose.observability.yml up -d
```
