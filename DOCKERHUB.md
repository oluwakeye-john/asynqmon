# Asynqmon with authentication

`oluwakeye/asynqmon` is a community-maintained, multi-platform build of [Asynqmon](https://github.com/hibiken/asynqmon), the web UI for monitoring and administering [Asynq](https://github.com/hibiken/asynq) task queues.

This image adds optional username/password authentication with a modern sign-in screen while retaining Asynqmon's queue, task, server, scheduler, Redis, and Prometheus views.

Source code: [github.com/oluwakeye-john/asynqmon](https://github.com/oluwakeye-john/asynqmon)

## Features

- Optional username/password authentication
- HTTP-only, 12-hour server-side sessions
- CSRF protection for state-changing requests
- Login rate limiting
- Light and dark themes
- Read-only mode
- Prometheus metrics support
- Images for `linux/amd64` and `linux/arm64`
- Runs as an unprivileged user in a minimal scratch image

## Quick start

Asynqmon requires access to the Redis instance used by Asynq.

On Docker Desktop for macOS or Windows, connect to Redis running on the host with:

```bash
docker run --rm \
  --name asynqmon \
  -p 8080:8080 \
  -e REDIS_ADDR=host.docker.internal:6379 \
  -e AUTH_USERNAME=operator \
  -e AUTH_PASSWORD='replace-with-a-long-random-password' \
  oluwakeye/asynqmon:v0.2.0
```

Open [http://localhost:8080](http://localhost:8080) and sign in with the configured credentials.

Use a versioned tag for deployments instead of `latest` so upgrades are explicit.

## Run with Redis in a Docker network

```bash
docker network create asynq-network

docker run -d \
  --name redis \
  --network asynq-network \
  redis:7-alpine

docker run --rm \
  --name asynqmon \
  --network asynq-network \
  -p 8080:8080 \
  -e REDIS_ADDR=redis:6379 \
  -e AUTH_USERNAME=operator \
  -e AUTH_PASSWORD='replace-with-a-long-random-password' \
  oluwakeye/asynqmon:v0.2.0
```

## Authentication

Authentication is enabled only when both variables are provided:

| Variable | Description |
| --- | --- |
| `AUTH_USERNAME` | Username required by the web UI and API |
| `AUTH_PASSWORD` | Password required by the web UI and API |

If both values are empty, authentication is disabled for backward compatibility. The container exits during startup if only one value is configured.

Sessions are stored in memory and expire after 12 hours. Run one replica when authentication is enabled; restarting the container signs active users out.

Terminate TLS at your ingress or reverse proxy in production. Asynqmon recognizes `X-Forwarded-Proto: https` and marks its session cookie as secure.

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP server port |
| `REDIS_ADDR` | `127.0.0.1:6379` | Redis address |
| `REDIS_URL` | empty | Redis or Sentinel connection URL |
| `REDIS_DB` | `0` | Redis database number |
| `REDIS_PASSWORD` | empty | Redis password |
| `REDIS_CLUSTER_NODES` | empty | Comma-separated Redis Cluster nodes |
| `REDIS_TLS` | empty | Server name used for Redis TLS validation |
| `REDIS_INSECURE_TLS` | `false` | Disable Redis TLS hostname validation |
| `READ_ONLY` | `false` | Disable state-changing queue and task operations |
| `MAX_PAYLOAD_LENGTH` | `200` | Maximum payload characters shown in the UI |
| `MAX_RESULT_LENGTH` | `200` | Maximum result characters shown in the UI |
| `ENABLE_METRICS_EXPORTER` | `false` | Expose queue metrics at `/metrics` |
| `PROMETHEUS_ADDR` | empty | Prometheus server queried by the metrics view |
| `AUTH_USERNAME` | empty | Sign-in username |
| `AUTH_PASSWORD` | empty | Sign-in password |

All options are also available as command-line flags:

```bash
docker run --rm oluwakeye/asynqmon:v0.2.0 --help
```

## Kubernetes

Keep credentials in a Kubernetes Secret rather than in the Deployment manifest:

```bash
kubectl create secret generic asynqmon-auth \
  --from-literal=username=operator \
  --from-literal=password='replace-with-a-long-random-password'
```

Inject the Secret into the container:

```yaml
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: asynqmon
          image: oluwakeye/asynqmon:v0.2.0
          ports:
            - name: http
              containerPort: 8080
          env:
            - name: REDIS_ADDR
              value: redis.default.svc.cluster.local:6379
            - name: AUTH_USERNAME
              valueFrom:
                secretKeyRef:
                  name: asynqmon-auth
                  key: username
            - name: AUTH_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: asynqmon-auth
                  key: password
```

For GitOps deployments, use External Secrets, Sealed Secrets, or another encrypted secret-management workflow. Do not commit plaintext credentials.

## Prometheus security note

When `ENABLE_METRICS_EXPORTER=true`, `/metrics` intentionally remains outside the application sign-in flow so Prometheus can scrape it. Restrict that endpoint using your cluster network policy, service topology, or ingress configuration.

## Tags and platforms

- Use version tags such as `v0.2.0` as stable deployment references.
- `latest` follows the most recently published version.
- Published manifests include `linux/amd64` and `linux/arm64`.

## License

Asynqmon is distributed under the [MIT License](https://github.com/oluwakeye-john/asynqmon/blob/master/LICENSE).
