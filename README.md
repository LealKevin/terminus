# Terminus

A multiplayer roguelike that runs in the terminal. The server is written in Go, players connect with a TUI client, walk around a shared map and fight goblins together.

I built it mainly to learn how a stateful game server behaves under load: concurrency in Go, keeping Postgres off the hot path, and running the whole thing on Kubernetes with Prometheus and Grafana watching it.

## How it works

- Clients talk to the server over plain TCP, one JSON message per line (`move`, `attack`, `getWorld`).
- Each connection gets its own goroutine. Player and mob state live in memory and are written to Postgres every 2 seconds, only for what changed.
- A game loop ticks every 500ms: it keeps 5 mobs on the map, moves them around and broadcasts their positions to everyone connected.
- The world layout is cached after the first load, so movement checks never hit the database.
- The server exposes Prometheus metrics on `:2112/metrics` (messages, errors, processing time, players online, moves and attacks).

## Layout

```
cmd/
  server/      game server
  client/      terminal client (Bubble Tea)
  botloader/   load testing bots
internal/
  domain/      player, mob, world rules
  app/         connection handling, game loop, mob manager, metrics
  client/      client state and rendering
  infra/       TCP server, Postgres (sqlc), in-memory stores
migrations/    SQL migrations
k8s/           kustomize manifests
grafana/       dashboards
```

## Running it

You need Go 1.24 and Docker.

```bash
cp .env.example .env
make up                      # postgres, migrations, server, prometheus, grafana
go run ./cmd/client          # in another terminal
```

Controls: `hjkl` or arrow keys to move, `yubn` for diagonals, `a` to attack, `q` to quit.

Grafana is on http://localhost:3000 and Prometheus on http://localhost:9090.

`make help` lists everything else (tests, migrations, docker image, k8s).

## Load testing

The botloader spawns bots that connect and move around randomly, ramping up, holding, then ramping down:

```bash
make bots BOTS=1000 RAMP=30 SUSTAIN=60
```

On the earlier AWS deployment, moving the world and mob state into memory brought move latency from around 20ms to 5ms and let the server handle 5000+ concurrent connections, up from about 350.

## Kubernetes

```bash
make k8s-secret          # creates k8s/db-secret.yaml from the example
make k8s-apply           # kubectl apply -k k8s/
make k8s-port-forward
```

This deploys Postgres as a StatefulSet, runs migrations as a Job, and starts the game server, Prometheus, Grafana and kube-state-metrics.

The game server runs as a single replica. State is held in memory per process, so scaling out would need sharding by world or a shared store, which is the next thing I'd like to work on.
