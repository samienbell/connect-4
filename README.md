# Connect 4

A self-contained, two-player (same screen) Connect 4 game.

| Layer          | Technology               |
|----------------|--------------------------|
| Frontend       | React (Vite), HTML, CSS  |
| Backend        | Go (REST API, `net/http`) |
| Database       | PostgreSQL 16            |
| Infrastructure | Docker, Kubernetes (Kustomize) |

The browser talks only to nginx, which serves the React build and proxies
`/api/*` to the Go backend. The backend stores every game and move in
PostgreSQL and applies the database schema itself on start-up. See
[docs/api.md](docs/api.md) for the API.

## Run with Docker Compose

```sh
docker compose up --build
```

Open http://localhost:3000.

## Local development (hot reload)

Prerequisites: Go 1.23+, Node 20+, Docker.

```sh
# 1. Database only
docker compose up -d db

# 2. Backend (http://localhost:8080)
cd backend
export DATABASE_URL="postgres://connect4:connect4@localhost:5432/connect4?sslmode=disable"
go mod tidy          # first time only: fetches deps and writes go.sum
go run ./cmd/server

# 3. Frontend (http://localhost:5173, proxies /api to :8080)
cd frontend
npm install
npm run dev
```

Run the game-logic tests with `cd backend && go test ./...`.

## Deploy to Kubernetes

```sh
# Local cluster (kind / minikube / Docker Desktop)
scripts/build-images.sh dev
kind load docker-image connect4-backend:dev connect4-frontend:dev   # or: minikube image load ...
kubectl apply -k deploy/k8s/overlays/dev
kubectl -n connect4-dev port-forward svc/frontend 3000:80
```

For production, see the notes at the top of
[deploy/k8s/overlays/prod/kustomization.yaml](deploy/k8s/overlays/prod/kustomization.yaml)
(registry, hostname, and creating the database secret).

## Directory structure

```
connect-4/
├── backend/
│   ├── cmd/server/          # main.go: config, DB connect, migrate, HTTP server
│   ├── internal/
│   │   ├── config/          # environment-variable settings
│   │   ├── game/            # rules: moves, win/draw detection (+ tests)
│   │   ├── handler/         # HTTP routes and JSON responses
│   │   └── store/           # PostgreSQL queries
│   └── migrations/          # embedded SQL schema, applied on start-up
├── frontend/
│   ├── index.html
│   ├── vite.config.js
│   └── src/
│       ├── api/             # fetch wrapper for the backend
│       ├── components/      # Board, Status
│       ├── hooks/           # useGame: game state + API calls
│       └── styles/          # app.css
├── deploy/
│   ├── docker/              # backend & frontend Dockerfiles, nginx.conf
│   └── k8s/
│       ├── base/            # Postgres StatefulSet, Deployments, Services, Ingress
│       └── overlays/        # dev (generated secret) and prod (replicas, registry)
├── scripts/build-images.sh
├── docs/api.md
├── docker-compose.yml
└── .env.example
```
