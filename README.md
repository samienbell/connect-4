# Connect 4

A self-contained Connect 4 game.

- **Accounts:** sign up with a name and password and sign in from any device.
  Passwords are stored as salted Argon2id hashes, and repeated wrong guesses
  are throttled.
- **Online play:** start a game and send the link to a friend. Moves appear
  live for both players, and anyone else with the link can watch.
- **Play on one device:** two people take turns on the same screen, no name needed.
- **Replays:** step through any game move by move, or let it play back.
- **Stats and leaderboard:** wins, losses, draws and win rate from finished online games.

| Layer          | Technology               |
|----------------|--------------------------|
| Frontend       | React (Vite), HTML, CSS  |
| Backend        | Go (REST API, `net/http`) |
| Database       | PostgreSQL 16            |
| Infrastructure | Docker, Kubernetes (Kustomize) |

The browser talks only to nginx, which serves the React build and proxies
`/api/*` to the Go backend. The backend stores players, games and every move
in PostgreSQL and applies database migrations itself on start-up. Live updates
use Server-Sent Events, with Postgres `LISTEN/NOTIFY` passing changes between
backend replicas. See [docs/api.md](docs/api.md) for the API.

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
go run ./cmd/server

# 3. Frontend (http://localhost:5173, proxies /api to :8080)
cd frontend
npm install
npm run dev
```

Run the backend tests with `cd backend && go test ./...`.

To try online play on one computer, create two accounts and sign in to the
second one in a private window (or another browser), then open the game link there.

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
│   │   ├── events/          # live-update hub (Postgres LISTEN/NOTIFY → SSE)
│   │   ├── game/            # rules: moves, win/draw detection, seats (+ tests)
│   │   ├── handler/         # HTTP routes, accounts & sessions, SSE stream
│   │   ├── password/        # salted Argon2id password hashing (+ tests)
│   │   └── store/           # PostgreSQL queries: games, moves, players, stats
│   └── migrations/          # versioned SQL files, applied once each on start-up
├── frontend/
│   ├── index.html
│   ├── vite.config.js
│   └── src/
│       ├── api/             # fetch wrapper + token storage
│       ├── components/      # Board, Players, AuthForm, PasswordForm, Leaderboard, ...
│       ├── hooks/           # usePlayer (account/session), useLiveGame (SSE)
│       ├── lib/             # seat/status text helpers
│       ├── views/           # Lobby, GameView, ReplayView
│       ├── router.jsx       # tiny client-side router
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
