# local-apps-manager-api

Gin API for discovering local API–WebUI pairs and controlling Docker stacks (inspired by `run-apps-on-local-docker`).

## Run

### Native (Postgres in Docker, API + WebUI on host)

1. Start Postgres: `docker compose up -d`
2. Copy `.env.example` → `.env`
3. `go run ./cmd/server`

### Docker dev (hot reload — Postgres + API + WebUI)

Full-repo bind mounts: Go changes reload via Air; Vue/config/public changes reload via Vite HMR.

```powershell
docker network create t3-net   # if missing
docker compose -f docker-compose.local.yml up --build
```

- WebUI: http://127.0.0.1:5195/apps
- API: http://127.0.0.1:8195/health

**Note:** Enable/Disable stack actions require Windows PowerShell and do not work from the Linux API container. Grid listing, auth, and discovery still work (GitHub root and Docker state are read-only mounts).

Default login: `armin` / `dopadopa123`

Dev ports: API **8195**, WebUI **5195**, Postgres **5455**

### Docker prod (Nginx — `pc-armin/local-app-manager`)

Production-like stack: Postgres + Go API + static WebUI behind Nginx. WebUI proxies `/api` and `/health` to the API container.

```powershell
docker network create t3-net   # if missing
docker compose -f docker-compose.prod.yml up --build -d
```

- WebUI (Nginx): http://127.0.0.1:5195/apps
- Images: `pc-armin/local-app-manager-api:latest`, `pc-armin/local-app-manager-webui:latest`
- Stack name: `local-app-manager`

**Note:** Enable/Disable actions still require Windows PowerShell on the host (API runs Linux in Docker). Grid listing, auth, and discovery work via mounted GitHub + devops plugin paths.

## Endpoints

- `GET /health`
- `POST /api/v1/auth/login`
- `GET /api/v1/apps` — grid data (`stack` = pair stem, `apiApp`, `webuiApp`, `apiInternalPort`, `webuiInternalPort`, `apiPort`, `webuiPort`, `externalHost`)
- `PATCH /api/v1/apps/:stem` — `{ "enabled": true|false }` starts/stops Docker stack
