# Connect 4

A self-contained Connect 4 game.

- **Accounts:** sign up with a name and password and sign in from any device.
  Passwords are stored as salted Argon2id hashes, and repeated wrong guesses
  are throttled.
- **Online play:** start a game and send the link to a friend. Moves appear
  live for both players, and anyone else with the link can watch.
- **Play on one device:** two people take turns on the same screen, no account needed.
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

## Prerequisites

To run the app you only need:

- **[Git](https://git-scm.com/downloads)**, to download the code.
- **[Docker Desktop](https://www.docker.com/products/docker-desktop/)** (Windows or
  macOS), or Docker Engine with the Compose plugin (Linux). Start it and wait until
  it says the engine is running before continuing.

Go and Node.js are built inside Docker, so you don't need them unless you want
to [develop locally](#local-development-hot-reload).

The commands below work in PowerShell, Command Prompt, macOS Terminal and Linux
shells unless marked otherwise.

## Get the code

```sh
git clone https://github.com/samienbell/connect-4.git
cd connect-4
```

## Run with Docker Compose

```sh
docker compose up --build
```

Open **http://localhost:3000** once the backend logs `"listening"` (the first
build takes a few minutes). Stop with `Ctrl+C`; `docker compose down -v` also
deletes the data.

If port 3000 or 5432 is taken, change the left-hand number under `ports:` in
[docker-compose.yml](docker-compose.yml).

## Run on Kubernetes

These steps use the Kubernetes cluster built into Docker Desktop.

1. **Turn on Kubernetes:** in Docker Desktop, go to **Settings → Kubernetes**,
   tick **Enable Kubernetes**, choose **kubeadm** if asked, then **Apply**. Wait
   until Docker Desktop shows Kubernetes as running. `kubectl get nodes` should
   then list one `Ready` node.

2. **Build the images:**

   ```sh
   docker build -f deploy/docker/backend.Dockerfile -t connect4-backend:dev .
   docker build -f deploy/docker/frontend.Dockerfile -t connect4-frontend:dev .
   ```

   Docker Desktop's cluster uses these local images directly.

3. **Deploy, and wait for everything to start:**

   ```sh
   kubectl apply -k deploy/k8s/overlays/dev
   kubectl -n connect4-dev rollout status statefulset/postgres --timeout=180s
   kubectl -n connect4-dev rollout status deployment/backend --timeout=180s
   kubectl -n connect4-dev rollout status deployment/frontend --timeout=180s
   ```

   Each command returns once that part is running. The backend may log
   "database not ready, retrying" once or twice while Postgres starts; that's expected.

4. **Open a tunnel to the app,** and leave this command running:

   ```sh
   kubectl -n connect4-dev port-forward svc/frontend 3001:80
   ```

   Open **http://localhost:3001**. Port 3001 is used so this can run alongside
   the Docker Compose version on port 3000. It has its own separate database.

To remove everything, including the cluster's database, run
`kubectl delete -k deploy/k8s/overlays/dev`.

**Using kind or minikube instead?** They can't see your local images. After
step 2, load them into the cluster with
`kind load docker-image connect4-backend:dev connect4-frontend:dev` or
`minikube image load connect4-backend:dev` (and the same for
`connect4-frontend:dev`), then continue with step 3.

For production, see the notes at the top of
[deploy/k8s/overlays/prod/kustomization.yaml](deploy/k8s/overlays/prod/kustomization.yaml)
(registry, hostname, creating the database secret). A public deployment also
needs HTTPS, because passwords are sent when people sign in.

## Local development (hot reload)

This runs the backend and frontend directly on your computer, so code changes
show up without rebuilding containers. You'll need **Go 1.23+** and
**Node.js 20+** in addition to Docker. Use three terminals, all starting in the
`connect-4` folder.

**Terminal 1, the database:**

```sh
docker compose up -d db
```

**Terminal 2, the backend** (http://localhost:8080):

PowerShell:

```powershell
cd backend
$env:DATABASE_URL = "postgres://connect4:connect4@localhost:5432/connect4?sslmode=disable"
go run ./cmd/server
```

macOS / Linux / Git Bash:

```sh
cd backend
export DATABASE_URL="postgres://connect4:connect4@localhost:5432/connect4?sslmode=disable"
go run ./cmd/server
```

**Terminal 3, the frontend** (http://localhost:5173, which forwards `/api` to the backend):

```sh
cd frontend
npm install
npm run dev
```

Open http://localhost:5173.

Run the backend tests with:

```sh
cd backend
go test ./...
```

To try online play on one computer, create two accounts and sign in to the
second one in a private window (or another browser), then open the game link there.

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
├── scripts/build-images.sh   # builds both images (bash; same as the docker build commands above)
├── docs/api.md
├── docker-compose.yml
└── .env.example
```
