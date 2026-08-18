# local-apps-manager-api

Gin API for discovering local API–WebUI pairs and controlling Docker stacks (inspired by `run-apps-on-local-docker`).

## Run

1. Start Postgres: `docker compose up -d`
2. Copy `.env.example` → `.env`
3. `go run ./cmd/server`

Default login: `armin` / `dopadopa123`

Dev ports: API **8195**, Postgres **5455**

## Endpoints

- `GET /health`
- `POST /api/v1/auth/login`
- `GET /api/v1/apps` — grid data (`stack` = pair stem, `apiApp`, `webuiApp`, `apiInternalPort`, `webuiInternalPort`, `apiPort`, `webuiPort`, `externalHost`)
- `PATCH /api/v1/apps/:stem` — `{ "enabled": true|false }` starts/stops Docker stack
