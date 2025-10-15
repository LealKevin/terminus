# Terminus

A high-performance multiplayer game server built with Go, featuring cloud-native deployment, monitoring, and auto-scaling capabilities.

## Overview

Terminus is a real-time multiplayer game server capable of handling 5000+ concurrent connections. Players navigate through a world grid, encounter mobs, and interact in real-time. The architecture demonstrates modern cloud deployment practices with comprehensive monitoring and performance optimization.

## Key Features

- **High Performance**: Optimized to handle 5000+ concurrent players with 5ms latency
- **Cloud Native**: Full Kubernetes deployment with auto-scaling on AWS EKS
- **Monitoring**: Complete observability stack with Prometheus and Grafana
- **Real-time Gaming**: WebSocket-based multiplayer with live combat and movement
- **Performance Optimization**: Strategic caching achieving 4x latency improvement
- **Load Testing**: Automated bot framework for performance validation

## Architecture

- **Server**: WebSocket-based game server with performance metrics
- **Client**: Terminal UI with Bubble Tea for interactive gameplay
- **Storage**: PostgreSQL with connection pooling and cached operations
- **Orchestration**: Kubernetes with StatefulSets and horizontal auto-scaling
- **Monitoring**: Prometheus metrics with Grafana dashboards
- **Cloud**: AWS EKS deployment with Network Load Balancers

## Technology Stack

- **Backend**: Go 1.24+, WebSockets, PostgreSQL
- **Container**: Docker with multi-platform builds
- **Orchestration**: Kubernetes, StatefulSets, HPA
- **Cloud**: AWS EKS, ECR, Network Load Balancer
- **Monitoring**: Prometheus, Grafana, kube-state-metrics
- **Performance**: Strategic caching, connection pooling

## Project Structure

```
terminus/
├── cmd/
│   ├── server/         # Game server with metrics
│   ├── client/         # Terminal client
│   └── botloader/      # Load testing framework
├── internal/
│   ├── app/           # Application handlers with caching
│   ├── client/        # Client-side logic
│   ├── domain/        # Business logic (Player, World, Mob)
│   └── infra/         # Infrastructure layer
│       ├── db/        # Database models and queries
│       ├── server/    # WebSocket server
│       └── store/     # Data persistence with caching
├── k8s/               # Kubernetes manifests
│   ├── aws/          # AWS EKS specific configs
│   └── minikube/     # Local development
└── docker-compose.yml
```

## Quick Start

### Local Development

```bash
docker-compose up
```

### Kubernetes Deployment

```bash
kubectl apply -f k8s/
```

### Load Testing

```bash
go run cmd/botloader/main.go -bots=1000 -duration=300s
```

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
DATABASE_URL=postgres://terminus:your_password@db:5432/terminus?sslmode=disable
WORLD_ID=123e4567-e89b-12d3-a456-426614174000
SERVER_HOST=your-loadbalancer-endpoint.elb.your-region.amazonaws.com
SERVER_PORT=4200
```

## Performance Achievements

- **Latency**: Reduced from 20ms to 5ms through caching optimization
- **Capacity**: Increased from 350 to 5000+ concurrent connections
- **Auto-scaling**: Kubernetes HPA based on CPU/memory metrics
- **Monitoring**: Real-time metrics with custom Grafana dashboards

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source and available under the MIT License.