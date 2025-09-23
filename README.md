# Terminus

A real-time multiplayer game server built with Go, featuring WebSocket connections and terminal-based clients.

## Overview

Terminus is a multiplayer game where players navigate through a world grid, encounter mobs, and interact in real-time. The game features automatic mob spawning, movement mechanics, and a clean separation between server and client components.

## Architecture

- **Server**: WebSocket-based game server with real-time player and mob updates
- **Client**: Terminal UI using Bubble Tea for interactive gameplay
- **Storage**: PostgreSQL with SQLC for type-safe database operations
- **Game Loop**: Automated mob spawning and movement every 500ms

## Features

- Real-time multiplayer gameplay
- Automatic mob spawning and AI movement
- Player movement and combat system
- WebSocket communication
- PostgreSQL persistence
- Docker containerization

## Project Structure

```
terminus/
├── cmd/
│   ├── server/         # Game server entrypoint
│   └── client/         # Terminal client
├── internal/
│   ├── app/           # Application handlers
│   ├── client/        # Client-side logic
│   ├── domain/        # Business logic (Player, World, Mob)
│   └── infra/         # Infrastructure layer
│       ├── db/        # Database models and queries
│       ├── server/    # WebSocket server
│       └── store/     # Data persistence layer
└── docker-compose.yml
```

## Getting Started

### Prerequisites

- Go 1.24+
- PostgreSQL
- Docker (optional)

### Running with Docker

```bash
docker-compose up
```

### Manual Setup

1. Start the server:
```bash
go run cmd/server/main.go
```

2. Connect with client:
```bash
go run cmd/client/main.go
```

## Database

The project uses SQLC for type-safe database operations. Models include:

- **Player**: Position, health, attack, defense stats
- **World**: Grid layout with dimensions
- **Mob**: AI-controlled entities with movement and combat

## Game Mechanics

- Players spawn randomly in valid world positions
- Mobs spawn automatically (max 5 per world)
- Movement uses 8-directional controls (N, S, E, W, NE, NW, SE, SW)
- Combat system with attack/defense calculations
- Real-time updates broadcast to all connected clients

## Development Roadmap

### Upcoming Features
- Kubernetes deployment configurations
- Prometheus metrics collection
- Grafana dashboards for monitoring
- Enhanced game mechanics
- Multi-world support

### Monitoring Stack
- **Prometheus**: Metrics collection for game statistics
- **Grafana**: Visualization dashboards
- **Kubernetes**: Container orchestration and scaling

## Configuration

Server runs on port 4200 by default. Database connection and other settings can be configured via environment variables.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source and available under the MIT License.